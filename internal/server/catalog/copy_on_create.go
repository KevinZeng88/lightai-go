package catalog

import (
	"fmt"
	"time"
)

// ============================================================================
// Copy-on-create operations between domain layers
// ============================================================================

// CreateNextLayerBundle deep-copies the parent's effective snapshot into a child bundle,
// adds the child's own ConfigSets, and stamps copy-on-create provenance on every item.
//
//   next_layer = deep_copy(parent.effective_snapshot) + own_sets + initial_local_edits
//
// The returned bundle has:
//   - InheritedBundleSnapshots populated from parent's effective snapshot (with snapshot stamps)
//   - OwnSets set to the provided ownSets
//   - LocalEdits initialized empty
//   - EffectiveView cleared (materialized on read)
func CreateNextLayerBundle(parent ConfigSetBundle, ownSets []ConfigSet, layer, id string) ConfigSetBundle {
	snapshotStamp := time.Now().UTC().Format(time.RFC3339)

	// Deep copy the effective snapshot from parent
	inherited := parent.DeepCopySnapshot(layer, id)

	// Stamp the snapshot timestamp on every inherited item
	for k, item := range inherited.Items {
		item.Snapshot_.CopiedAt = snapshotStamp
		// Inherited items: schema/snapshot fields are immutable at this layer.
		// Value/state fields may be modified by this layer via local edits.
		// We reset state checked/enabled to false for inherited items unless
		// they are required/system_generated — the child layer must explicitly
		// enable what it wants to override.
		if !item.Schema.Required {
			item.State_.Checked = false
		}
		// Ensure editable is true unless schema.readonly
		if !item.Schema.ReadOnly {
			item.State_.Editable = true
		}
		inherited.Items[k] = item
	}

	// Collect inherited item keys for owner preservation in own sets
	inheritedKeys := make(map[string]ConfigItem)
	for k, v := range inherited.Items {
		inheritedKeys[k] = v
	}

	// Preserve inherited schema.owner and snapshot in own set items
	for i, set := range ownSets {
		for k, item := range set.Items {
			item.AlignTiers() // ensure tiered fields are populated from flat fields
			if inherited, exists := inheritedKeys[k]; exists {
				// Inherited item: preserve schema.owner and snapshot
				if item.Schema.Owner == "" || item.Schema.Owner != inherited.Schema.Owner {
					item.Schema.Owner = inherited.Schema.Owner
					item.Schema.OwnerLayer = inherited.Schema.OwnerLayer
				}
				item.Snapshot_ = inherited.Snapshot_
			} else {
				// New item genuinely owned by this layer
				if item.Schema.Owner == "" {
					item.Schema.Owner = set.ConfigSetKey
				}
				if item.Schema.OwnerLayer == "" {
					item.Schema.OwnerLayer = set.ConfigSetKey
				}
			}
			set.Items[k] = item
		}
		ownSets[i] = set
	}

	return ConfigSetBundle{
		InheritedBundleSnapshots: []ConfigSet{inherited},
		OwnSets:                  ownSets,
		LocalEdits:               make(map[string]map[string]ConfigItemLocalEdit),
	}
}

// ============================================================================
// Local edits
// ============================================================================

// ApplyLocalEdit records a value/state override on an inherited or own item.
// Returns an error if the edit targets an immutable schema field.
func ApplyLocalEdit(bundle *ConfigSetBundle, configSetKey, itemKey string, value any, enabled, checked *bool, reason, editedBy string) error {
	if bundle == nil {
		return fmt.Errorf("bundle is nil")
	}
	if itemKey == "" {
		return fmt.Errorf("item_key is required")
	}

	// Find the item in any layer (inherited or own) to validate the edit is permitted
	item := bundle.findItem(itemKey)
	if item == nil {
		return fmt.Errorf("item %q not found in bundle", itemKey)
	}

	// Enforce schema/snapshot read-only: cannot modify schema fields via local edit
	if item.Schema.ReadOnly {
		return fmt.Errorf("item %q is schema.read_only — cannot modify via local edit", itemKey)
	}
	if !item.State_.Editable {
		return fmt.Errorf("item %q is not editable in current layer", itemKey)
	}

	if configSetKey == "" {
		configSetKey = item.Schema.ConfigSetKey
	}

	if bundle.LocalEdits == nil {
		bundle.LocalEdits = make(map[string]map[string]ConfigItemLocalEdit)
	}
	if bundle.LocalEdits[configSetKey] == nil {
		bundle.LocalEdits[configSetKey] = make(map[string]ConfigItemLocalEdit)
	}

	edit := ConfigItemLocalEdit{
		ConfigSetKey: configSetKey,
		ItemKey:      itemKey,
		Reason:       reason,
		EditedAt:     time.Now().UTC().Format(time.RFC3339),
		EditedBy:     editedBy,
	}

	if value != nil {
		edit.LocalValue = value
	}
	if enabled != nil {
		edit.Enabled = enabled
	}
	if checked != nil {
		edit.Checked = checked
	}

	bundle.LocalEdits[configSetKey][itemKey] = edit
	return nil
}

// findItem looks up an item by key across all layers in the bundle.
func (b *ConfigSetBundle) findItem(itemKey string) *ConfigItem {
	// Check own sets first (most specific)
	for _, set := range b.OwnSets {
		if item, ok := set.Items[itemKey]; ok {
			return &item
		}
	}
	// Check inherited snapshots
	for _, snap := range b.InheritedBundleSnapshots {
		if item, ok := snap.Items[itemKey]; ok {
			return &item
		}
	}
	return nil
}

// ============================================================================
// Copy-on-create validation
// ============================================================================

// ValidateSchemaImmutability checks that no inherited item has had its
// schema or snapshot fields modified by the current layer.
// Returns a list of violations if any immutable fields were changed.
func ValidateSchemaImmutability(bundle ConfigSetBundle) []string {
	var violations []string

	inheritedKeys := make(map[string]ConfigItem)
	for _, snap := range bundle.InheritedBundleSnapshots {
		for k, v := range snap.Items {
			inheritedKeys[k] = v
		}
	}

	for _, set := range bundle.OwnSets {
		for k, v := range set.Items {
			inherited, exists := inheritedKeys[k]
			if !exists {
				continue // new own item, not inherited
			}
			// Schema must not be redefined
			if v.Schema.Key != inherited.Schema.Key {
				violations = append(violations, fmt.Sprintf("item %q: schema.key changed from %q to %q", k, inherited.Schema.Key, v.Schema.Key))
			}
			if v.Schema.Owner != inherited.Schema.Owner {
				violations = append(violations, fmt.Sprintf("item %q: schema.owner changed from %q to %q (owner must not change)", k, inherited.Schema.Owner, v.Schema.Owner))
			}
			// Snapshot must not be changed
			if v.Snapshot_.FromLayer != inherited.Snapshot_.FromLayer || v.Snapshot_.FromID != inherited.Snapshot_.FromID {
				violations = append(violations, fmt.Sprintf("item %q: snapshot metadata changed", k))
			}
		}
	}

	return violations
}

// AddOwnSet appends an own ConfigSet to the bundle and aligns tiers on all its items.
// For items that already exist in inherited snapshots, only value/state fields are
// applied; schema.owner and snapshot fields from inherited items are preserved.
func (b *ConfigSetBundle) AddOwnSet(cs ConfigSet) {
	// Collect inherited item keys for reference
	inheritedItems := make(map[string]ConfigItem)
	for _, snap := range b.InheritedBundleSnapshots {
		for k, v := range snap.Items {
			inheritedItems[k] = v
		}
	}

	for k, item := range cs.Items {
		item.AlignTiers()
		// If this item is inherited, preserve schema owner and snapshot from inheritance
		if inherited, exists := inheritedItems[k]; exists {
			if item.Schema.Owner == "" || item.Schema.Owner != inherited.Schema.Owner {
				item.Schema.Owner = inherited.Schema.Owner
				item.Schema.OwnerLayer = inherited.Schema.OwnerLayer
			}
			// Snapshot must remain as set by copy-on-create
			if item.Snapshot_.FromLayer == "" {
				item.Snapshot_ = inherited.Snapshot_
			}
		} else {
			// New item genuinely owned by this layer's ConfigSet
			if item.Schema.Owner == "" {
				item.Schema.Owner = cs.ConfigSetKey
			}
			if item.Schema.OwnerLayer == "" {
				item.Schema.OwnerLayer = cs.ConfigSetKey
			}
		}
		cs.Items[k] = item
	}
	b.OwnSets = append(b.OwnSets, cs)
}
