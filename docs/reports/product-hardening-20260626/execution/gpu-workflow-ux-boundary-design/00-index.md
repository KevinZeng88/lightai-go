# LightAI Go GPU / Model Workflow UX Boundary Design Package

Created: 2026-06-26  
Purpose: provide a complete product/UX design, implementation plan, and validation contract for the GPU/model runtime management workflow before Claude modifies code.

## Why this package exists

The current Web UI exposes too many internal objects and internal configuration keys. Users are being asked to understand `BackendRuntime`, `ConfigSet`, `launcher.command`, `{{MODEL_CONTAINER_PATH}}`, raw IDs, and catalog internals. That is the wrong mental model.

This package defines the product-level workflow and the implementation boundaries so Claude can understand and execute a coherent repair instead of making isolated Vue patches.

## Documents

1. `01-product-boundaries-and-user-mental-model.md`  
   Defines the three lines: Model line, Runtime line, Deployment line. This is the conceptual foundation.

2. `02-current-ux-problems-and-root-causes.md`  
   Lists observed issues, why they happen, and the root cause pattern.

3. `03-target-ux-design-by-page.md`  
   Defines the target user experience page by page: Runtime Templates, Node Runtime Configs, Model Library, Model Deployments, Model Instances.

4. `04-implementation-plan.md`  
   Concrete engineering steps, recommended file changes, data/view-model mapping, and commit plan.

5. `05-validation-and-acceptance.md`  
   Functional, UI, API, regression, E2E, and evidence requirements.

6. `06-claude-understanding-check-prompt.md`  
   Prompt for Claude to read these documents and summarize understanding before coding.

7. `07-claude-autorun-prompt-after-approval.md`  
   Prompt to use only after Claude's understanding is approved.

## Non-goals for this round

- OpenAI Gateway
- API Key management
- Usage metering
- Billing
- Multi-tenant quota redesign
- Kubernetes/Ray scheduler
- Historical compatibility migration

## Core principle

Users configure a runtime environment; they do not edit internal ConfigSet structures.

