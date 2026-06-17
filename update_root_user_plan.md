1. **Implement `UpdateUser` in `UserRepository`**
   - Add `UpdateUser(input models.User, password *string) (bool, error)` to `server/internal/infrastructure/db/user-repository.go`.
   - The method should update the user record, and if a password is provided, generate a new hash and update it.

2. **Create `UpdateRootUserCommand`**
   - Create `server/internal/usecase/command/auth/update-root-user-command.go`.
   - The command will take the new `Username` and/or `Password`.
   - It will fetch the root user, update its fields, and save via the `UserRepository.UpdateUser` method.

3. **Add `UpdateRootUser` method to `AuthBO`**
   - Add a method to `server/internal/usecase/business-logic/auth-bo.go` that executes the `UpdateRootUserCommand`.

4. **Add `AuthUpdateHandler` to `AdminController`**
   - Implement `AuthUpdateHandler` in `server/internal/infrastructure/server/rest/auth/auth-controller.go`.
   - It should accept a `PUT` request with new credentials.
   - It will verify the current user is authorized before allowing changes.

5. **Register Route in `routes.go`**
   - Add a `PUT /auth/update` route in `server/internal/infrastructure/server/rest/routes.go` under the authenticated routes section.

6. **Pre-commit Checks**
   - Ensure proper testing, verification, review, and reflection are done by calling pre commit instructions.
