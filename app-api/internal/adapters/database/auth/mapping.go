package authdbrepo

import (
	"prasankit-api/internal/modules/auth"
)

func accountToModel(a auth.Account) accountModel {
	return accountModel{
		ID:                a.ID,
		PrimaryEmail:      a.PrimaryEmail,
		DisplayName:       a.DisplayName,
		AccountStatusCode: a.AccountStatusCode,
		FailedLoginCount:  a.FailedLoginCount,
		LockedUntil:       a.LockedUntil,
		LastLoginAt:       a.LastLoginAt,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
		DeletedAt:         a.DeletedAt,
	}
}

func modelToAccount(m accountModel) auth.Account {
	return auth.Account{
		ID:                m.ID,
		PrimaryEmail:      m.PrimaryEmail,
		DisplayName:       m.DisplayName,
		AccountStatusCode: m.AccountStatusCode,
		FailedLoginCount:  m.FailedLoginCount,
		LockedUntil:       m.LockedUntil,
		LastLoginAt:       m.LastLoginAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		DeletedAt:         m.DeletedAt,
	}
}

func identityToModel(i auth.Identity) identityModel {
	return identityModel{
		ID:                i.ID,
		UserAccountID:     i.UserAccountID,
		IdentityTypeCode:  i.IdentityTypeCode,
		Email:             i.Email,
		EmailVerifiedAt:   i.EmailVerifiedAt,
		PasswordHash:      i.PasswordHash,
		PasswordChangedAt: i.PasswordChangedAt,
		LastUsedAt:        i.LastUsedAt,
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
		DeletedAt:         i.DeletedAt,
	}
}

func modelToIdentity(m identityModel) auth.Identity {
	return auth.Identity{
		ID:                m.ID,
		UserAccountID:     m.UserAccountID,
		IdentityTypeCode:  m.IdentityTypeCode,
		Email:             m.Email,
		EmailVerifiedAt:   m.EmailVerifiedAt,
		PasswordHash:      m.PasswordHash,
		PasswordChangedAt: m.PasswordChangedAt,
		LastUsedAt:        m.LastUsedAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		DeletedAt:         m.DeletedAt,
	}
}

func emailVerificationTokenToModel(t auth.EmailVerificationToken) emailVerificationTokenModel {
	return emailVerificationTokenModel{
		ID:             t.ID,
		AuthIdentityID: t.AuthIdentityID,
		TokenHash:      t.TokenHash,
		ExpiresAt:      t.ExpiresAt,
		UsedAt:         t.UsedAt,
		RevokedAt:      t.RevokedAt,
		CreatedAt:      t.CreatedAt,
	}
}

func modelToEmailVerificationToken(m emailVerificationTokenModel) auth.EmailVerificationToken {
	return auth.EmailVerificationToken{
		ID:             m.ID,
		AuthIdentityID: m.AuthIdentityID,
		TokenHash:      m.TokenHash,
		ExpiresAt:      m.ExpiresAt,
		UsedAt:         m.UsedAt,
		RevokedAt:      m.RevokedAt,
		CreatedAt:      m.CreatedAt,
	}
}

func passwordResetTokenToModel(t auth.PasswordResetToken) passwordResetTokenModel {
	return passwordResetTokenModel{
		ID:             t.ID,
		AuthIdentityID: t.AuthIdentityID,
		TokenHash:      t.TokenHash,
		ExpiresAt:      t.ExpiresAt,
		UsedAt:         t.UsedAt,
		RevokedAt:      t.RevokedAt,
		CreatedAt:      t.CreatedAt,
	}
}

func modelToPasswordResetToken(m passwordResetTokenModel) auth.PasswordResetToken {
	return auth.PasswordResetToken{
		ID:             m.ID,
		AuthIdentityID: m.AuthIdentityID,
		TokenHash:      m.TokenHash,
		ExpiresAt:      m.ExpiresAt,
		UsedAt:         m.UsedAt,
		RevokedAt:      m.RevokedAt,
		CreatedAt:      m.CreatedAt,
	}
}
