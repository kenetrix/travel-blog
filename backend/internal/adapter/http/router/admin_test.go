package router_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminAndPasswordGate(t *testing.T) {
	h, accounts := newStack(t, false)
	file := filepath.Join(t.TempDir(), "initial-admin.txt")
	if _, err := accounts.EnsureAdmin(file); err != nil {
		t.Fatal(err)
	}
	creds, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var username, password string
	for _, line := range strings.Split(string(creds), "\n") {
		if v, ok := strings.CutPrefix(line, "username: "); ok {
			username = v
		}
		if v, ok := strings.CutPrefix(line, "password: "); ok {
			password = v
		}
	}

	// The generated password only gets the account, the change and logout
	rec := do(t, h, "POST", "/api/auth/login", `{"username":"`+username+`","password":"`+password+`"}`, nil)
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	admin := map[string]string{"Cookie": sessionCookie(t, rec)}
	rec = do(t, h, "GET", "/api/user", "", admin)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"mustChangePassword":true`) || !strings.Contains(rec.Body.String(), `"role":"admin"`) {
		t.Fatalf("account: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/api/admin/users", "", admin); rec.Code != 403 {
		t.Errorf("admin API before the change: %d", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/countries/DEU", `{"status":"visited"}`, admin); rec.Code != 403 {
		t.Errorf("write before the change: %d", rec.Code)
	}
	rec = do(t, h, "PUT", "/api/user/password", `{"currentPassword":"`+password+`","newPassword":"a brand new password"}`, admin)
	if rec.Code != 204 {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body)
	}

	// Now the admin API works, for administrators only
	rec = do(t, h, "GET", "/api/admin/users", "", admin)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"username":"admin"`) {
		t.Fatalf("list users: %d %s", rec.Code, rec.Body)
	}
	alice := register(t, h, "alice")
	if rec := do(t, h, "GET", "/api/admin/users", "", alice); rec.Code != 403 {
		t.Errorf("list users as user: %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/admin/settings", "", alice); rec.Code != 403 {
		t.Errorf("settings as user: %d", rec.Code)
	}

	rec = do(t, h, "PUT", "/api/admin/settings", `{"registrationOpen":false}`, admin)
	if rec.Code != 204 {
		t.Fatalf("close registration: %d %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "GET", "/api/auth/users/newcomer", "", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"registrationOpen":false`) {
		t.Errorf("username check: %d %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "POST", "/api/auth/register", `{"username":"newcomer","password":"correct horse battery"}`, nil)
	if rec.Code != 403 {
		t.Errorf("register while closed: %d", rec.Code)
	}
}
