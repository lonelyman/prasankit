package authrepo

import (
	"context"
	"errors"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/pkg/dbtypes"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrNotImplemented = errors.New("auth repository method is not implemented")

type Repository struct {
	db *gorm.DB
}

type userAccountRow struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid"`
	PrimaryEmail     string     `gorm:"column:primary_email"`
	Status           string     `gorm:"column:status"`
	LastLoginAt      *time.Time `gorm:"column:last_login_at"`
	FailedLoginCount int        `gorm:"column:failed_login_count"`
	LockedUntil      *time.Time `gorm:"column:locked_until"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	DeletedAt        *time.Time `gorm:"column:deleted_at"`
	DeletedBy        *uuid.UUID `gorm:"column:deleted_by"`
}

func (userAccountRow) TableName() string {
	return "user_accounts"
}

type authIdentityRow struct {
	ID                uuid.UUID  `gorm:"column:id;type:uuid"`
	UserAccountID     uuid.UUID  `gorm:"column:user_account_id"`
	IdentityType      string     `gorm:"column:identity_type"`
	Provider          string     `gorm:"column:provider"`
	ProviderUserID    *string    `gorm:"column:provider_user_id"`
	Email             string     `gorm:"column:email"`
	EmailVerifiedAt   *time.Time `gorm:"column:email_verified_at"`
	PasswordHash      string     `gorm:"column:password_hash"`
	PasswordChangedAt *time.Time `gorm:"column:password_changed_at"`
	LastUsedAt        *time.Time `gorm:"column:last_used_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at"`
}

func (authIdentityRow) TableName() string {
	return "auth_identities"
}

type emailVerificationTokenRow struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid"`
	AuthIdentityID uuid.UUID  `gorm:"column:auth_identity_id"`
	TokenHash      string     `gorm:"column:token_hash"`
	Status         string     `gorm:"column:status"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	ExpiresAt      time.Time  `gorm:"column:expires_at"`
	UsedAt         *time.Time `gorm:"column:used_at"`
}

func (emailVerificationTokenRow) TableName() string {
	return "auth_email_verification_tokens"
}

type passwordResetTokenRow struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid"`
	AuthIdentityID uuid.UUID  `gorm:"column:auth_identity_id"`
	TokenHash      string     `gorm:"column:token_hash"`
	Status         string     `gorm:"column:status"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	ExpiresAt      time.Time  `gorm:"column:expires_at"`
	UsedAt         *time.Time `gorm:"column:used_at"`
}

func (passwordResetTokenRow) TableName() string {
	return "auth_password_reset_tokens"
}

type authSessionRow struct {
	ID             uuid.UUID     `gorm:"column:id;type:uuid"`
	UserAccountID  uuid.UUID     `gorm:"column:user_account_id"`
	SessionKeyHash string        `gorm:"column:session_key_hash"`
	Status         string        `gorm:"column:status"`
	IPAddress      *string       `gorm:"column:ip_address"`
	UserAgent      string        `gorm:"column:user_agent"`
	DeviceLabel    string        `gorm:"column:device_label"`
	CreatedAt      time.Time     `gorm:"column:created_at"`
	LastSeenAt     *time.Time    `gorm:"column:last_seen_at"`
	ExpiresAt      time.Time     `gorm:"column:expires_at"`
	RevokedAt      *time.Time    `gorm:"column:revoked_at"`
	RevokedReason  string        `gorm:"column:revoked_reason"`
	MetadataJSON   dbtypes.JSONB `gorm:"column:metadata_json;type:jsonb"`
}

func (authSessionRow) TableName() string {
	return "auth_sessions"
}

type loginAttemptRow struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid"`
	UserAccountID *uuid.UUID `gorm:"column:user_account_id"`
	Email         string     `gorm:"column:email"`
	Success       bool       `gorm:"column:success"`
	FailureReason string     `gorm:"column:failure_reason"`
	IPAddress     string     `gorm:"column:ip_address"`
	UserAgent     string     `gorm:"column:user_agent"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (loginAttemptRow) TableName() string {
	return "auth_login_attempts"
}

type securityEventRow struct {
	ID            uuid.UUID     `gorm:"column:id;type:uuid"`
	UserAccountID *uuid.UUID    `gorm:"column:user_account_id"`
	EventType     string        `gorm:"column:event_type"`
	Severity      string        `gorm:"column:severity"`
	IPAddress     *string       `gorm:"column:ip_address"`
	UserAgent     string        `gorm:"column:user_agent"`
	MetadataJSON  dbtypes.JSONB `gorm:"column:metadata_json;type:jsonb"`
	CreatedAt     time.Time     `gorm:"column:created_at"`
}

func (securityEventRow) TableName() string {
	return "security_events"
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) WithinTransaction(ctx context.Context, fn func(context.Context, auth.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRepository(tx))
	})
}

func (r *Repository) FindUserAccountByID(ctx context.Context, id uuid.UUID) (*auth.UserAccount, error) {
	var row userAccountRow
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrUserAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) FindUserAccountByEmail(ctx context.Context, email string) (*auth.UserAccount, error) {
	var row userAccountRow
	err := r.db.WithContext(ctx).
		Table("user_accounts").
		Select("user_accounts.*").
		Joins("JOIN auth_identities ON auth_identities.user_account_id = user_accounts.id").
		Where("auth_identities.identity_type = ?", string(auth.AuthIdentityTypeEmailPassword)).
		Where("auth_identities.provider = ?", string(auth.AuthProviderEmail)).
		Where("auth_identities.email = ?", email).
		Where("auth_identities.deleted_at IS NULL").
		Where("user_accounts.deleted_at IS NULL").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrUserAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) FindAuthIdentityByID(ctx context.Context, id uuid.UUID) (*auth.AuthIdentity, error) {
	var row authIdentityRow
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrAuthIdentityNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) FindAuthIdentityByEmail(ctx context.Context, email string) (*auth.AuthIdentity, error) {
	var row authIdentityRow
	err := r.db.WithContext(ctx).
		Where("identity_type = ?", string(auth.AuthIdentityTypeEmailPassword)).
		Where("provider = ?", string(auth.AuthProviderEmail)).
		Where("email = ?", email).
		Where("deleted_at IS NULL").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrAuthIdentityNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) FindEmailVerificationTokenByHash(ctx context.Context, tokenHash string) (*auth.EmailVerificationToken, error) {
	var row emailVerificationTokenRow
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrEmailVerificationTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) FindPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*auth.PasswordResetToken, error) {
	var row passwordResetTokenRow
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrPasswordResetTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) CreateUserAccount(ctx context.Context, account *auth.UserAccount) error {
	if err := ensureUUID(&account.ID); err != nil {
		return err
	}
	if account.Status == "" {
		account.Status = auth.UserAccountStatusPendingVerification
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now().UTC()
	}
	if account.UpdatedAt.IsZero() {
		account.UpdatedAt = account.CreatedAt
	}

	row := userAccountRow{
		ID:               account.ID,
		PrimaryEmail:     account.PrimaryEmail,
		Status:           string(account.Status),
		LastLoginAt:      account.LastLoginAt,
		FailedLoginCount: account.FailedLoginCount,
		LockedUntil:      account.LockedUntil,
		CreatedAt:        account.CreatedAt,
		UpdatedAt:        account.UpdatedAt,
		DeletedAt:        account.DeletedAt,
		DeletedBy:        account.DeletedBy,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	account.ID = row.ID
	account.CreatedAt = row.CreatedAt
	account.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) CreateAuthIdentity(ctx context.Context, identity *auth.AuthIdentity) error {
	if err := ensureUUID(&identity.ID); err != nil {
		return err
	}
	if identity.IdentityType == "" {
		identity.IdentityType = auth.AuthIdentityTypeEmailPassword
	}
	if identity.Provider == "" {
		identity.Provider = auth.AuthProviderEmail
	}
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = time.Now().UTC()
	}
	if identity.UpdatedAt.IsZero() {
		identity.UpdatedAt = identity.CreatedAt
	}

	row := authIdentityRow{
		ID:                identity.ID,
		UserAccountID:     identity.UserAccountID,
		IdentityType:      string(identity.IdentityType),
		Provider:          string(identity.Provider),
		ProviderUserID:    stringPtrOrNil(identity.ProviderUserID),
		Email:             identity.Email,
		EmailVerifiedAt:   identity.EmailVerifiedAt,
		PasswordHash:      identity.PasswordHash,
		PasswordChangedAt: identity.PasswordChangedAt,
		LastUsedAt:        identity.LastUsedAt,
		CreatedAt:         identity.CreatedAt,
		UpdatedAt:         identity.UpdatedAt,
		DeletedAt:         identity.DeletedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return mapCreateAuthIdentityError(err)
	}

	identity.ID = row.ID
	identity.CreatedAt = row.CreatedAt
	identity.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) RevokeActiveEmailVerificationTokens(ctx context.Context, authIdentityID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&emailVerificationTokenRow{}).
		Where("auth_identity_id = ?", authIdentityID).
		Where("status = ?", string(auth.EmailVerificationTokenStatusActive)).
		Update("status", string(auth.EmailVerificationTokenStatusRevoked)).
		Error
}

func (r *Repository) CreateEmailVerificationToken(ctx context.Context, token *auth.EmailVerificationToken) error {
	if err := ensureUUID(&token.ID); err != nil {
		return err
	}
	if token.Status == "" {
		token.Status = auth.EmailVerificationTokenStatusActive
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	if token.ExpiresAt.IsZero() {
		return auth.ErrEmailVerificationTokenExpiresAtRequired
	}

	row := emailVerificationTokenRow{
		ID:             token.ID,
		AuthIdentityID: token.AuthIdentityID,
		TokenHash:      token.TokenHash,
		Status:         string(token.Status),
		CreatedAt:      token.CreatedAt,
		ExpiresAt:      token.ExpiresAt,
		UsedAt:         token.UsedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	token.ID = row.ID
	token.CreatedAt = row.CreatedAt
	return nil
}

func (r *Repository) RevokeActivePasswordResetTokens(ctx context.Context, authIdentityID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&passwordResetTokenRow{}).
		Where("auth_identity_id = ?", authIdentityID).
		Where("status = ?", string(auth.PasswordResetTokenStatusActive)).
		Update("status", string(auth.PasswordResetTokenStatusRevoked)).
		Error
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, token *auth.PasswordResetToken) error {
	if err := ensureUUID(&token.ID); err != nil {
		return err
	}
	if token.Status == "" {
		token.Status = auth.PasswordResetTokenStatusActive
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	if token.ExpiresAt.IsZero() {
		return auth.ErrPasswordResetTokenExpiresAtRequired
	}

	row := passwordResetTokenRow{
		ID:             token.ID,
		AuthIdentityID: token.AuthIdentityID,
		TokenHash:      token.TokenHash,
		Status:         string(token.Status),
		CreatedAt:      token.CreatedAt,
		ExpiresAt:      token.ExpiresAt,
		UsedAt:         token.UsedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	token.ID = row.ID
	token.CreatedAt = row.CreatedAt
	return nil
}

func (r *Repository) MarkEmailVerificationTokenUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&emailVerificationTokenRow{}).
		Where("id = ?", id).
		Where("status = ?", string(auth.EmailVerificationTokenStatusActive)).
		Updates(map[string]any{
			"status":  string(auth.EmailVerificationTokenStatusUsed),
			"used_at": usedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrEmailVerificationTokenNotFound
	}
	return nil
}

func (r *Repository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&passwordResetTokenRow{}).
		Where("id = ?", id).
		Where("status = ?", string(auth.PasswordResetTokenStatusActive)).
		Updates(map[string]any{
			"status":  string(auth.PasswordResetTokenStatusUsed),
			"used_at": usedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrPasswordResetTokenNotFound
	}
	return nil
}

func (r *Repository) MarkAuthIdentityEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&authIdentityRow{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"email_verified_at": verifiedAt,
			"updated_at":        verifiedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrAuthIdentityNotFound
	}
	return nil
}

func (r *Repository) UpdateAuthIdentityPassword(ctx context.Context, id uuid.UUID, passwordHash string, changedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&authIdentityRow{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"password_hash":       passwordHash,
			"password_changed_at": changedAt,
			"updated_at":          changedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrAuthIdentityNotFound
	}
	return nil
}

func (r *Repository) ActivateUserAccount(ctx context.Context, id uuid.UUID, updatedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&userAccountRow{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"status":     string(auth.UserAccountStatusActive),
			"updated_at": updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrUserAccountNotFound
	}
	return nil
}

func (r *Repository) UpdateUserAccountLoginSuccess(ctx context.Context, id uuid.UUID, loggedInAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&userAccountRow{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"last_login_at":      loggedInAt,
			"failed_login_count": 0,
			"locked_until":       nil,
			"updated_at":         loggedInAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrUserAccountNotFound
	}
	return nil
}

func (r *Repository) MarkAuthIdentityLastUsed(ctx context.Context, id uuid.UUID, lastUsedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&authIdentityRow{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"last_used_at": lastUsedAt,
			"updated_at":   lastUsedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrAuthIdentityNotFound
	}
	return nil
}

func (r *Repository) FindActiveAuthSessionByHash(ctx context.Context, sessionKeyHash string) (*auth.AuthSession, error) {
	var row authSessionRow
	err := r.db.WithContext(ctx).
		Where("session_key_hash = ?", sessionKeyHash).
		Where("status = ?", string(auth.AuthSessionStatusActive)).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrAuthSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) ListActiveAuthSessionsByUserAccountID(ctx context.Context, userAccountID uuid.UUID) ([]auth.AuthSession, error) {
	var rows []authSessionRow
	err := r.db.WithContext(ctx).
		Where("user_account_id = ?", userAccountID).
		Where("status = ?", string(auth.AuthSessionStatusActive)).
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	sessions := make([]auth.AuthSession, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, *row.toDomain())
	}
	return sessions, nil
}

func (r *Repository) CreateAuthSession(ctx context.Context, session *auth.AuthSession) error {
	if err := ensureUUID(&session.ID); err != nil {
		return err
	}
	if session.Status == "" {
		session.Status = auth.AuthSessionStatusActive
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if session.ExpiresAt.IsZero() {
		return auth.ErrAuthSessionExpiresAtRequired
	}
	if session.MetadataJSON == nil {
		session.MetadataJSON = map[string]any{}
	}

	row := authSessionRow{
		ID:             session.ID,
		UserAccountID:  session.UserAccountID,
		SessionKeyHash: session.SessionKeyHash,
		Status:         string(session.Status),
		IPAddress:      stringPtrOrNil(session.IPAddress),
		UserAgent:      session.UserAgent,
		DeviceLabel:    session.DeviceLabel,
		CreatedAt:      session.CreatedAt,
		LastSeenAt:     session.LastSeenAt,
		ExpiresAt:      session.ExpiresAt,
		RevokedAt:      session.RevokedAt,
		RevokedReason:  session.RevokedReason,
		MetadataJSON:   dbtypes.NewJSONB(session.MetadataJSON),
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	session.ID = row.ID
	session.CreatedAt = row.CreatedAt
	return nil
}

func (r *Repository) RevokeAuthSessionByHash(ctx context.Context, sessionKeyHash string, revokedAt time.Time, reason string) error {
	result := r.db.WithContext(ctx).
		Model(&authSessionRow{}).
		Where("session_key_hash = ?", sessionKeyHash).
		Where("status = ?", string(auth.AuthSessionStatusActive)).
		Updates(map[string]any{
			"status":         string(auth.AuthSessionStatusRevoked),
			"revoked_at":     revokedAt,
			"revoked_reason": reason,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return auth.ErrAuthSessionNotFound
	}
	return nil
}

func (r *Repository) RevokeActiveAuthSessionsByUserAccountID(ctx context.Context, userAccountID uuid.UUID, revokedAt time.Time, reason string) error {
	return r.db.WithContext(ctx).
		Model(&authSessionRow{}).
		Where("user_account_id = ?", userAccountID).
		Where("status = ?", string(auth.AuthSessionStatusActive)).
		Updates(map[string]any{
			"status":         string(auth.AuthSessionStatusRevoked),
			"revoked_at":     revokedAt,
			"revoked_reason": reason,
		}).
		Error
}

func (r *Repository) CreateLoginAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	if err := ensureUUID(&attempt.ID); err != nil {
		return err
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	row := loginAttemptRow{
		ID:            attempt.ID,
		UserAccountID: attempt.UserAccountID,
		Email:         attempt.Email,
		Success:       attempt.Success,
		FailureReason: attempt.FailureReason,
		IPAddress:     attempt.IPAddress,
		UserAgent:     attempt.UserAgent,
		CreatedAt:     attempt.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	attempt.ID = row.ID
	attempt.CreatedAt = row.CreatedAt
	return nil
}

func (r *Repository) CreateSecurityEvent(ctx context.Context, event *auth.SecurityEvent) error {
	if err := ensureUUID(&event.ID); err != nil {
		return err
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Severity == "" {
		event.Severity = auth.SecurityEventSeverityInfo
	}
	if event.MetadataJSON == nil {
		event.MetadataJSON = map[string]any{}
	}

	row := securityEventRow{
		ID:            event.ID,
		UserAccountID: event.UserAccountID,
		EventType:     event.EventType,
		Severity:      string(event.Severity),
		IPAddress:     stringPtrOrNil(event.IPAddress),
		UserAgent:     event.UserAgent,
		MetadataJSON:  dbtypes.NewJSONB(event.MetadataJSON),
		CreatedAt:     event.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	return nil
}

func ensureUUID(id *uuid.UUID) error {
	if *id != uuid.Nil {
		return nil
	}
	newID, err := ids.NewUUID()
	if err != nil {
		return err
	}
	*id = newID
	return nil
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func mapCreateAuthIdentityError(err error) error {
	if isUniqueViolation(err, "uq_auth_identities_email_password_active") {
		return auth.ErrEmailAlreadyRegistered
	}
	return err
}

func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return strings.Contains(err.Error(), `unique constraint "`+constraintName+`"`)
	}
	if pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == constraintName || strings.Contains(err.Error(), `unique constraint "`+constraintName+`"`)
}

func (r userAccountRow) toDomain() *auth.UserAccount {
	return &auth.UserAccount{
		ID:               r.ID,
		PrimaryEmail:     r.PrimaryEmail,
		Status:           auth.UserAccountStatus(r.Status),
		LastLoginAt:      r.LastLoginAt,
		FailedLoginCount: r.FailedLoginCount,
		LockedUntil:      r.LockedUntil,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		DeletedAt:        r.DeletedAt,
		DeletedBy:        r.DeletedBy,
	}
}

func (r authIdentityRow) toDomain() *auth.AuthIdentity {
	providerUserID := ""
	if r.ProviderUserID != nil {
		providerUserID = *r.ProviderUserID
	}
	return &auth.AuthIdentity{
		ID:                r.ID,
		UserAccountID:     r.UserAccountID,
		IdentityType:      auth.AuthIdentityType(r.IdentityType),
		Provider:          auth.AuthProvider(r.Provider),
		ProviderUserID:    providerUserID,
		Email:             r.Email,
		EmailVerifiedAt:   r.EmailVerifiedAt,
		PasswordHash:      r.PasswordHash,
		PasswordChangedAt: r.PasswordChangedAt,
		LastUsedAt:        r.LastUsedAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		DeletedAt:         r.DeletedAt,
	}
}

func (r emailVerificationTokenRow) toDomain() *auth.EmailVerificationToken {
	return &auth.EmailVerificationToken{
		ID:             r.ID,
		AuthIdentityID: r.AuthIdentityID,
		TokenHash:      r.TokenHash,
		Status:         auth.EmailVerificationTokenStatus(r.Status),
		CreatedAt:      r.CreatedAt,
		ExpiresAt:      r.ExpiresAt,
		UsedAt:         r.UsedAt,
	}
}

func (r passwordResetTokenRow) toDomain() *auth.PasswordResetToken {
	return &auth.PasswordResetToken{
		ID:             r.ID,
		AuthIdentityID: r.AuthIdentityID,
		TokenHash:      r.TokenHash,
		Status:         auth.PasswordResetTokenStatus(r.Status),
		CreatedAt:      r.CreatedAt,
		ExpiresAt:      r.ExpiresAt,
		UsedAt:         r.UsedAt,
	}
}

func (r authSessionRow) toDomain() *auth.AuthSession {
	ipAddress := ""
	if r.IPAddress != nil {
		ipAddress = *r.IPAddress
	}
	return &auth.AuthSession{
		ID:             r.ID,
		UserAccountID:  r.UserAccountID,
		SessionKeyHash: r.SessionKeyHash,
		Status:         auth.AuthSessionStatus(r.Status),
		IPAddress:      ipAddress,
		UserAgent:      r.UserAgent,
		DeviceLabel:    r.DeviceLabel,
		CreatedAt:      r.CreatedAt,
		LastSeenAt:     r.LastSeenAt,
		ExpiresAt:      r.ExpiresAt,
		RevokedAt:      r.RevokedAt,
		RevokedReason:  r.RevokedReason,
		MetadataJSON:   map[string]any(r.MetadataJSON),
	}
}
