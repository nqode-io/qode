# Notes

- Clarify where to define knownContainerDirs so both workspace and detect packages can share it. **Answer**: All known dirs should be included:
   - Turborepo / Nx: `apps/`, `packages/`, `libs/`
   - Lerna: `packages/`
   - Rush: `apps/`, `libraries/`
   - Custom: `services/`, `projects/`