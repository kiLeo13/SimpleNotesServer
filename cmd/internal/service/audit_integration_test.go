package service

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"simplenotes/cmd/internal/contract"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/domain/policy"
	"simplenotes/cmd/internal/domain/sqlite/repository"
	cognitoclient "simplenotes/cmd/internal/infrastructure/aws/cognito"
	"simplenotes/cmd/internal/utils"
	"simplenotes/cmd/internal/utils/validators"
)

type sequenceAuditIDGenerator struct {
	next int64
}

func (s *sequenceAuditIDGenerator) NextID() (int64, error) {
	s.next++
	return s.next, nil
}

type noopGateway struct{}

func (noopGateway) PostToConnection(context.Context, string, interface{}) error { return nil }
func (noopGateway) DeleteConnection(context.Context, string) error              { return nil }

type noopS3 struct{}

func (noopS3) UploadFile([]byte, string) error { return nil }
func (noopS3) DeleteFile(string) error         { return nil }

type fakeCognitoClient struct{}

func (fakeCognitoClient) SignUp(*cognitoclient.User) (string, error) { return "", nil }
func (fakeCognitoClient) SignIn(*cognitoclient.UserLogin) (*cognitoclient.AuthCreate, error) {
	return nil, nil
}
func (fakeCognitoClient) GlobalSignOut(string) error                           { return nil }
func (fakeCognitoClient) ConfirmAccount(*cognitoclient.UserConfirmation) error { return nil }
func (fakeCognitoClient) ResendConfirmation(string) error                      { return nil }
func (fakeCognitoClient) AdminDeleteUser(string) error                         { return nil }

type fakeLookupClient struct {
	company *entity.Company
	err     error
}

func (f *fakeLookupClient) GetByCNPJ(context.Context, string) (*entity.Company, error) {
	if f.err != nil {
		return nil, f.err
	}
	clone := *f.company
	return &clone, nil
}

func TestNoteUpdateAuditGroupsMultipleFieldChanges(t *testing.T) {
	db := newTestDB(t)
	validate := newTestValidator()

	auditRepo := repository.NewAuditRepository(db)
	auditSvc := newTestAuditService(t, db, 1000)
	userRepo := repository.NewUserRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	connRepo := repository.NewConnectionRepository(db)
	wsSvc := NewWebSocketService(connRepo, noopGateway{})
	noteSvc := NewNoteService(db, noteRepo, userRepo, wsSvc, noopS3{}, validate, auditSvc, policy.NewNotePolicy())

	actor := &entity.User{
		Username:    "editor",
		Email:       "editor@example.com",
		Permissions: entity.PermissionEditNotes.Add(entity.PermissionSeeHiddenNotes),
		Active:      true,
		CreatedAt:   utils.NowUTC(),
		UpdatedAt:   utils.NowUTC(),
	}
	if err := userRepo.Save(actor); err != nil {
		t.Fatalf("save actor: %v", err)
	}

	note := &entity.Note{
		Name:        "Old Name",
		Content:     "hello",
		CreatedByID: actor.ID,
		Tags:        "alpha beta",
		NoteType:    entity.NoteTypeMarkdown,
		ContentSize: 5,
		Visibility:  entity.VisibilityPublic,
		CreatedAt:   utils.NowUTC(),
		UpdatedAt:   utils.NowUTC(),
	}
	if err := noteRepo.Save(note); err != nil {
		t.Fatalf("save note: %v", err)
	}

	newName := "New Name"
	newVisibility := string(entity.VisibilityPrivate)
	req := &contract.UpdateNoteRequest{
		Name:       &newName,
		Visibility: &newVisibility,
		Tags:       []string{"gamma", "delta"},
	}

	resp, apierr := noteSvc.UpdateNote(actor, note.ID, req)
	if apierr != nil {
		t.Fatalf("update note returned api error: %#v", apierr)
	}
	if resp == nil {
		t.Fatal("expected note response")
	}

	events, err := auditRepo.List(&repository.AuditLogFilter{
		Limit:      10,
		ActionType: auditActionPtr(entity.AuditActionNoteUpdate),
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 note update audit event, got %d", len(events))
	}
	if events[0].SubjectID != strconv.Itoa(note.ID) {
		t.Fatalf("unexpected subject id: %s", events[0].SubjectID)
	}
	if len(events[0].Changes) != 3 {
		t.Fatalf("expected 3 grouped changes, got %d", len(events[0].Changes))
	}

	fields := map[string]bool{}
	for _, change := range events[0].Changes {
		fields[change.FieldName] = true
		if change.EventID != events[0].ID {
			t.Fatalf("change event id %d does not match parent %d", change.EventID, events[0].ID)
		}
	}

	for _, field := range []string{"name", "visibility", "tags"} {
		if !fields[field] {
			t.Fatalf("missing audit change for field %s", field)
		}
	}
}

func TestDeleteUserCreatesAuditEvent(t *testing.T) {
	db := newTestDB(t)

	auditRepo := repository.NewAuditRepository(db)
	auditSvc := newTestAuditService(t, db, 2000)
	userRepo := repository.NewUserRepository(db)
	connRepo := repository.NewConnectionRepository(db)
	wsSvc := NewWebSocketService(connRepo, noopGateway{})
	userSvc := NewUserService(db, userRepo, newTestValidator(), wsSvc, fakeCognitoClient{}, auditSvc, policy.NewUserPolicy())

	actor := &entity.User{
		Username:    "moderator",
		Email:       "mod@example.com",
		Permissions: entity.PermissionDeleteUsers,
		Active:      true,
		CreatedAt:   utils.NowUTC(),
		UpdatedAt:   utils.NowUTC(),
	}
	target := &entity.User{
		Username:    "target",
		Email:       "target@example.com",
		Permissions: 0,
		Active:      true,
		CreatedAt:   utils.NowUTC(),
		UpdatedAt:   utils.NowUTC(),
	}
	if err := userRepo.Save(actor); err != nil {
		t.Fatalf("save actor: %v", err)
	}
	if err := userRepo.Save(target); err != nil {
		t.Fatalf("save target: %v", err)
	}

	if apierr := userSvc.DeleteUser(actor, strconv.Itoa(target.ID)); apierr != nil {
		t.Fatalf("delete user returned api error: %#v", apierr)
	}

	deletedUser, err := userRepo.FindByID(target.ID)
	if err != nil {
		t.Fatalf("find deleted user: %v", err)
	}
	if deletedUser == nil || deletedUser.Active {
		t.Fatal("expected user to be soft deleted")
	}

	events, err := auditRepo.List(&repository.AuditLogFilter{
		Limit:      10,
		ActionType: auditActionPtr(entity.AuditActionUserDelete),
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 user delete audit event, got %d", len(events))
	}
	if len(events[0].Changes) != 1 {
		t.Fatalf("expected 1 delete change, got %d", len(events[0].Changes))
	}
	change := events[0].Changes[0]
	if change.FieldName != "active" {
		t.Fatalf("unexpected change field: %s", change.FieldName)
	}
	if change.OldValue == nil || *change.OldValue != "true" {
		t.Fatalf("unexpected old value: %v", change.OldValue)
	}
	if change.NewValue == nil || *change.NewValue != "false" {
		t.Fatalf("unexpected new value: %v", change.NewValue)
	}
}

func TestGetCompanyByCNPJCreatesAuditEvent(t *testing.T) {
	db := newTestDB(t)

	auditRepo := repository.NewAuditRepository(db)
	auditSvc := newTestAuditService(t, db, 3000)
	companyRepo := repository.NewCompanyRepository(db)
	userRepo := repository.NewUserRepository(db)
	miscSvc := NewMiscService(&fakeLookupClient{
		company: &entity.Company{
			CNPJ:      "12345678000195",
			LegalName: "Magalu Teste",
		},
	}, companyRepo, auditSvc)

	actor := &entity.User{
		ID:          99,
		Username:    "lookup",
		Email:       "lookup@example.com",
		Permissions: entity.PermissionPerformLookup,
		Active:      true,
		CreatedAt:   utils.NowUTC(),
		UpdatedAt:   utils.NowUTC(),
	}
	if err := userRepo.Save(actor); err != nil {
		t.Fatalf("save actor: %v", err)
	}

	resp, apierr := miscSvc.GetCompanyByCNPJ(actor, "12345678000195")
	if apierr != nil {
		t.Fatalf("lookup returned api error: %#v", apierr)
	}
	if resp == nil || resp.CNPJ == "" {
		t.Fatal("expected company response")
	}

	events, err := auditRepo.List(&repository.AuditLogFilter{
		Limit:      10,
		ActionType: auditActionPtr(entity.AuditActionCompanyLookup),
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 company lookup audit event, got %d", len(events))
	}
	if events[0].SubjectID != "12345678000195" {
		t.Fatalf("unexpected company subject id: %s", events[0].SubjectID)
	}
	if len(events[0].Changes) != 2 {
		t.Fatalf("expected 2 lookup attributes, got %d", len(events[0].Changes))
	}
}

func TestAuditServiceGetAuditLogsPaginatesByBeforeID(t *testing.T) {
	db := newTestDB(t)
	auditSvc := newTestAuditService(t, db, 4000)

	for i := range 3 {
		event := &entity.AuditLogEvent{
			ActorUserID: intPtr(1),
			ActionType:  entity.AuditActionNoteCreate,
			SubjectType: entity.AuditSubjectNote,
			SubjectID:   strconv.Itoa(i + 1),
			Changes: []*entity.AuditLogChange{
				newAuditCreateValue("name", entity.AuditValueTypeString, "note"),
			},
		}
		if err := auditSvc.Record(nil, event); err != nil {
			t.Fatalf("record audit event: %v", err)
		}
	}

	actor := &entity.User{
		Permissions: entity.PermissionManageUsers,
	}

	page1, apierr := auditSvc.GetAuditLogs(actor, &contract.AuditLogListRequest{Limit: 2})
	if apierr != nil {
		t.Fatalf("get audit logs page 1: %#v", apierr)
	}
	if len(page1.Entries) != 2 {
		t.Fatalf("expected 2 entries on page 1, got %d", len(page1.Entries))
	}
	if page1.NextBeforeID == nil {
		t.Fatal("expected next_before_id on page 1")
	}

	beforeID, err := strconv.ParseInt(*page1.NextBeforeID, 10, 64)
	if err != nil {
		t.Fatalf("parse next_before_id: %v", err)
	}

	page2, apierr := auditSvc.GetAuditLogs(actor, &contract.AuditLogListRequest{
		Limit:    2,
		BeforeID: &beforeID,
	})
	if apierr != nil {
		t.Fatalf("get audit logs page 2: %#v", apierr)
	}
	if len(page2.Entries) != 1 {
		t.Fatalf("expected 1 entry on page 2, got %d", len(page2.Entries))
	}
}

func newTestAuditService(t *testing.T, db *gorm.DB, startID int64) *AuditService {
	t.Helper()

	auditSvc, err := NewAuditService(db, repository.NewAuditRepository(db), &sequenceAuditIDGenerator{next: startID})
	if err != nil {
		t.Fatalf("new audit service: %v", err)
	}
	return auditSvc
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_fk=1"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&entity.AuditLogEvent{},
		&entity.AuditLogChange{},
		&entity.Note{},
		&entity.User{},
		&entity.Connection{},
		&entity.Company{},
		&entity.CompanyPartner{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	return db
}

func newTestValidator() *validator.Validate {
	validate := validator.New()
	_ = validate.RegisterValidation("hasupper", validators.HasUpper)
	_ = validate.RegisterValidation("haslower", validators.HasLower)
	_ = validate.RegisterValidation("hasdigit", validators.HasDigit)
	_ = validate.RegisterValidation("hasspecial", validators.HasSpecial)
	_ = validate.RegisterValidation("nodupes", validators.NoDupes)
	_ = validate.RegisterValidation("nospaces", validators.NoWhiteSpaces)
	return validate
}

func auditActionPtr(action entity.AuditActionType) *entity.AuditActionType {
	return &action
}

func intPtr(value int) *int {
	return &value
}

var _ CompanyLookupClient = (*fakeLookupClient)(nil)
var _ cognitoclient.Client = fakeCognitoClient{}
