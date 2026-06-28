# Credits Disabled E2E Tests

This package contains temporary pre-credit to credit upgrade compatibility tests.

The default e2e suite runs with credits enabled. Tests in this package exercise
behavior that must remain valid for installations that started before credits
were enabled, or that still run with credits disabled during the upgrade window.

Remove this package once credits are fully ready and credits-disabled deployments
are no longer supported by the e2e contract.
