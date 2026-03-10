---
name: krangka-add-feature
description: Step-by-step guide for adding new features to krangka applications following hexagonal architecture. TDD-first with user review gates: write service tests, pause for user confirmation, then implement. Covers migration, domain, ports, repository, DTOs, bootstrap, Swagger. Use when adding a new CRUD feature, entity, or API resource to a krangka project. For architecture rules see krangka-hexagonal; for dependency wiring see krangka-dependency-wiring.
---

# Adding New Features in Krangka

This skill guides adding a new feature (e.g. User management) to a krangka application. Follow the steps in order. **TDD is mandatory**: write all tests first, then **pause for user review** before implementing.

## TDD Review Gate (Critical)

After writing service tests (Step 6):

1. **Stop** — do not proceed to implementation
2. **Present** the test scenarios to the user (list scenarios covered per method/endpoint)
3. **Ask**: "Please review the test scenarios above. Are they correct and complete? Reply to confirm or request changes."
4. **Wait** for user confirmation before implementing
5. **Proceed** to implementation only after user confirms

Never skip the review gate. Implementation without confirmed test scenarios is an anti-pattern.

**Presenting scenarios**: List each service method with its test cases (e.g. "GetUserByID: success, not found, repo error"). Keep it scannable so the user can quickly verify coverage.

## Canonical Order

```
Task Progress:
- [ ] 1. Database Migration
- [ ] 2. Domain (entity + filter)
- [ ] 3. Failure definitions
- [ ] 4. Port interfaces (inbound + outbound)
- [ ] 5. Generate mocks
- [ ] 6. Service tests (TDD) — all scenarios first
- [ ] 6b. ⏸️ REVIEW GATE — present scenarios, wait for user confirmation
- [ ] 7. Implement service
- [ ] 8. Register service in bootstrap
- [ ] 9. Database repository
- [ ] 10. Register repository
- [ ] 11. DTOs
- [ ] 12. HTTP handler
- [ ] 13. Register handler
- [ ] 14. Swagger (godoc + make swag)
- [ ] 15. Run migration
```

> ⚠️ **TDD**: Write all test scenarios first (red), pause for user review, then implement (green) only after confirmation. See krangka-engineering-principles.

## Quick Reference

### Commands

```bash
make migrate-new repo=mariadb name=create_table_users
make mock
make swag
make migrate-up repo=mariadb
```

### File Locations

| Artifact | Path |
|----------|------|
| Migration | `internal/adapter/outbound/mariadb/migrations/scripts/` |
| Domain | `internal/core/domain/<entity>.go` |
| Failures | `shared/failure/failure.go` |
| Inbound port | `internal/core/port/inbound/<entity>.go` |
| Outbound port | `internal/core/port/outbound/repositories/<entity>.go` |
| Service | `internal/core/service/<entity>/service.go` |
| Repository impl | `internal/adapter/outbound/mariadb/repositories/<entity>.go` |
| DTOs | `internal/adapter/inbound/http/handler/dto/<entity>.go` |
| Handler | `internal/adapter/inbound/http/handler/<entity>.go` |

## Step-by-Step Summary

### 1. Migration

Create table with `id`, `created_at`, `updated_at`, `deleted_at`. Use soft delete (`deleted_at int DEFAULT 0`).

### 2. Domain

Plain structs only. Use `qwery` tags for DB mapping. Include `DeletedAt int`. Add `Filter` struct for list operations (optional filters use `*bool`).

### 3. Failures

Add typed errors in `shared/failure/failure.go`. Convention: `HTTPSTATUS + sequential number` (e.g. `404003`, `409003`).

### 4. Ports

- **Inbound**: Service interface (use case contract)
- **Outbound**: Repository interface in `repositories/`
- Add `//go:generate mockgen` to each port file
- Add `Get<Entity>Repository()` to main `Repository` interface

### 5. Mocks

Run `make mock`. Mocks go to `internal/mocks/inbound/` and `internal/mocks/outbound/repositories/`.

### 6–7. Service (TDD)

**6. Tests first**: Table-driven tests for every method. Cover: success, not found (`sql.ErrNoRows` → typed failure), repo error propagation. **Stop here.** Present scenarios to user and wait for confirmation.

**7. Implementation** (only after user confirms): Call `tracer.Trace(ctx)` + `defer span.End()` in every method. Use `fail.Wrap(err)`. Map `sql.ErrNoRows` to typed failures. See krangka-fail.

### 8. Register Service

Add `service<Entity> Resource[*entity.Service]` to `Dependency`. Create getter `GetService<Entity>(repo outbound.Repository)`. Service getters accept `repo` so handlers use `GetqweryMain()`, workers use `GetqweryWorker()`. See krangka-dependency-wiring.

### 9. Repository

Use `RunRaw()` with inline SQL. qwery template syntax: `{{ .field }}`. Always `tracer.Trace(ctx)`, `fail.Wrap(err)`. Use `WithPagination()` + `WithOrderBy()` for list. `WHERE deleted_at = 0` on SELECT/UPDATE. Optional filters: `{{ if .field }}`.

### 10. Register Repository

Add to `mariaDBRepository` struct and constructor. Implement getter with transaction-aware pattern: if `qweryTx != nil`, return repo bound to tx.

### 11. DTOs

- Request: `uri`, `query`, `json` tags (Fiber v3: `uri` for path params); `Validate()`; `Transform()` → domain
- Response: `json` tags; `Transform(domain)` → DTO
- Use `ulid.Make().String()` for create ID
- `swaggerignore:"true"` on path params (`uri` tag)
- Never expose `DeletedAt` in responses

### 12. HTTP Handler

Use `c.Bind().URI()` for path params, `c.Bind().Query()` for query params, `c.Bind().Body()` for body. Use `c.Context()` for request context. Call `req.Validate()`; call service; `response.SuccessOK` / `SuccessCreated` / `SuccessOKWithPagination`. Wrap parse/validation errors with `fail.Wrap(err).WithFailure(fail.ErrBadRequest)`. Propagate service errors directly.

### 13. Register Handler

Add handler to `GetHTTPHandlers()` in `cmd/bootstrap/dependency.go`. Pass `GetService<Entity>(repo)` where `repo := d.GetRepository(d.GetqweryMain())`.

### 14. Swagger

Every handler method needs godoc: `@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`. Run `make swag`.

### 15. Run Migration

`make migrate-up repo=mariadb`

**Optional**: If the feature publishes events and you need to consume them, add a subscriber handler in `internal/adapter/inbound/subscriber/handler/` and register in `GetSubscriberHandlers()`. See krangka-subscriber.

## Service Test Example

Use **table-driven tests** with mocks. Cover success, not found (`sql.ErrNoRows` → typed failure), and repo error propagation:

```go
// internal/core/service/user/service_test.go
func TestUserService_GetUserByID(t *testing.T) {
	tests := []struct {
		scenario    string
		id          string
		setup       func(*mocks.MockRepository, *mocksrepos.MockUser)
		wantErr     bool
		wantFailure *fail.Failure
	}{
		{
			scenario: "success",
			id:       "01JKM1234",
			setup: func(repo *mocks.MockRepository, userRepo *mocksrepos.MockUser) {
				user := &domain.User{ID: "01JKM1234", Username: "testuser"}
				repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
				userRepo.EXPECT().GetUserByID(gomock.Any(), "01JKM1234").Return(user, nil).Times(1)
			},
			wantErr: false,
		},
		{
			scenario: "not found",
			id:       "01JKM1234",
			setup: func(repo *mocks.MockRepository, userRepo *mocksrepos.MockUser) {
				repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
				userRepo.EXPECT().GetUserByID(gomock.Any(), "01JKM1234").Return(nil, sql.ErrNoRows).Times(1)
			},
			wantErr:     true,
			wantFailure: failure.ErrUserNotFound,
		},
		{
			scenario: "repo error",
			id:       "01JKM1234",
			setup: func(repo *mocks.MockRepository, userRepo *mocksrepos.MockUser) {
				repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
				userRepo.EXPECT().GetUserByID(gomock.Any(), "01JKM1234").
					Return(nil, errors.New("db error")).Times(1)
			},
			wantErr:     true,
			wantFailure: fail.ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockRepository(ctrl)
			mockUserRepo := mocksrepos.NewMockUser(ctrl)
			tt.setup(mockRepo, mockUserRepo)

			svc := NewService(nil, nil, mockRepo, nil)
			result, err := svc.GetUserByID(context.Background(), tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				if tt.wantFailure != nil {
					assert.True(t, fail.IsFailure(err, tt.wantFailure))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.id, result.ID)
		})
	}
}
```

## Cross-References

- **krangka-hexagonal**: Layer rules, import boundaries, test structure
- **krangka-fail**: Error wrapping, `fail.Wrap`, typed failures
- **krangka-engineering-principles**: TDD, correctness over speed
- **krangka-dependency-wiring**: Service/repo getters, bootstrap pattern
- **krangka-pagination**: List endpoints, `WithPagination`, `WithOrderBy`
- **krangka-subscriber**: Event handlers (optional, when feature publishes events)

## Full Documentation

For complete code examples (domain, ports, service, repository, DTOs, handler, service tests), see [.krangka/docs/04_adding-new-features.md](.krangka/docs/04_adding-new-features.md).
