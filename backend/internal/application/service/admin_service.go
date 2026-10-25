package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"travel-blog/backend/internal/domain"
)

// passwordAlphabet omits easily confused characters (0/O, 1/l/I).
const passwordAlphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomPassword(n int) (string, error) {
	out := make([]byte, n)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for i := range out {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = passwordAlphabet[v.Int64()]
	}
	return string(out), nil
}

// EnsureAdmin creates the first administrator if there is none: user "admin" with a random password
// that must change at first login, written to file (operator-readable) until then. created reports it.
func (s *AccountService) EnsureAdmin(file string) (created bool, err error) {
	s.initialAdminFile = file
	n, err := s.users.CountAdmins()
	if err != nil || n > 0 {
		return false, err
	}

	username := "admin"
	if existing, err := s.users.GetByUsername(username); err != nil {
		return false, err
	} else if existing != nil {
		// The name belongs to a regular account already; we do not take it over
		username = "admin_" + uuid.NewString()[:4]
	}
	password, err := randomPassword(20)
	if err != nil {
		return false, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.hashCost)
	if err != nil {
		return false, err
	}
	// File first, so the password is never lost
	content := fmt.Sprintf("username: %s\npassword: %s\n", username, password)
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		return false, fmt.Errorf("write initial admin credentials: %w", err)
	}
	u := domain.User{
		ID:                 domain.UserID(uuid.NewString()),
		Role:               domain.RoleAdmin,
		MustChangePassword: true,
		Profile: domain.Profile{
			Username:    username,
			DisplayName: "Administrator",
			Language:    "en",
			Timezone:    "UTC",
			CreatedAt:   s.now().UTC(),
		},
	}
	if err := s.users.Create(u, string(hash)); err != nil {
		os.Remove(file)
		return false, err
	}
	return true, nil
}

func (s *AccountService) RegistrationOpen() (bool, error) {
	return s.settings.RegistrationOpen()
}

func (s *AccountService) requireAdmin(actor domain.UserID) error {
	u, err := s.users.Get(actor)
	if err != nil {
		return err
	}
	if u == nil || u.Role != domain.RoleAdmin {
		return fmt.Errorf("%w: administrators only", ErrForbidden)
	}
	return nil
}

// target returns a manageable account: it must exist and cannot be
// the local user.
func (s *AccountService) target(id domain.UserID) (*domain.User, error) {
	if id == domain.LocalUserID {
		return nil, fmt.Errorf("%w: the local user cannot be managed", ErrInvalid)
	}
	u, err := s.users.Get(id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("%w: user %s", ErrNotFound, id)
	}
	return u, nil
}

func (s *AccountService) Users(actor domain.UserID) ([]domain.User, error) {
	if err := s.requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.users.List()
}

// SetRole changes the role of an account. The last administrator
// stays an administrator.
func (s *AccountService) SetRole(actor, id domain.UserID, role domain.Role) (domain.User, error) {
	if err := s.requireAdmin(actor); err != nil {
		return domain.User{}, err
	}
	if role != domain.RoleUser && role != domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("%w: role must be user or admin", ErrInvalid)
	}
	u, err := s.target(id)
	if err != nil {
		return domain.User{}, err
	}
	if u.Role == domain.RoleAdmin && role != domain.RoleAdmin {
		n, err := s.users.CountAdmins()
		if err != nil {
			return domain.User{}, err
		}
		if n <= 1 {
			return domain.User{}, fmt.Errorf("%w: there must be at least one administrator", ErrConflict)
		}
	}
	if err := s.users.SetRole(id, role); err != nil {
		return domain.User{}, err
	}
	u.Role = role
	return *u, nil
}

// ResetPassword sets a new password, ends all sessions of the account and requires its own
// password at the next login. Change your own password with ChangePassword.
func (s *AccountService) ResetPassword(actor, id domain.UserID, password string) error {
	if err := s.requireAdmin(actor); err != nil {
		return err
	}
	if actor == id {
		return fmt.Errorf("%w: change your own password in the profile", ErrInvalid)
	}
	if _, err := s.target(id); err != nil {
		return err
	}
	if err := checkPassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.hashCost)
	if err != nil {
		return err
	}
	if err := s.users.SetPasswordHash(id, string(hash)); err != nil {
		return err
	}
	if err := s.users.SetMustChangePassword(id, true); err != nil {
		return err
	}
	return s.sessions.DeleteOthers(id, "")
}

// DeleteUser removes another account with all data; delete your own
// with DeleteAccount.
func (s *AccountService) DeleteUser(actor, id domain.UserID) error {
	if err := s.requireAdmin(actor); err != nil {
		return err
	}
	if actor == id {
		return fmt.Errorf("%w: delete your own account in the profile", ErrInvalid)
	}
	if _, err := s.target(id); err != nil {
		return err
	}
	return s.DeleteAccount(id)
}

// AdminSettings returns whether self-registration is open
// (administrators only).
func (s *AccountService) AdminSettings(actor domain.UserID) (registrationOpen bool, err error) {
	if err := s.requireAdmin(actor); err != nil {
		return false, err
	}
	return s.settings.RegistrationOpen()
}

func (s *AccountService) SetRegistrationOpen(actor domain.UserID, open bool) error {
	if err := s.requireAdmin(actor); err != nil {
		return err
	}
	return s.settings.SetRegistrationOpen(open)
}
