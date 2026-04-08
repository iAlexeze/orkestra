package auth

import (
	"html/template"
	"net/http"
	"os"

	cc "github.com/orkspace/orkestra-cc/cc"
)

var (
	loginTpl = template.Must(template.ParseFS(cc.Assets, "assets/templates/login.html"))

	ork           = "orkestra"
	username      = os.Getenv("ADMIN_USERNAME")
	password      = os.Getenv("ADMIN_PASSWORD")
	sessionSecret = []byte(os.Getenv("SESSION_SECRET"))
	orkSession    = "orkestra_session"
)

func applyDefaults() {
	if username == "" {
		username = ork
	}
	if password == "" {
		password = ork
	}
	if sessionSecret == nil {
		sessionSecret = []byte("dev-secret")
	}
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	loginTpl.Execute(w, nil)
}

func LoginPost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	user := r.Form.Get("username")
	pass := r.Form.Get("password")

	applyDefaults()

	if user != username || pass != password {
		w.WriteHeader(http.StatusUnauthorized)
		loginTpl.Execute(w, map[string]string{"Error": "Invalid credentials"})
		return
	}

	token := signSession(user)

	http.SetCookie(w, &http.Cookie{
		Name:     orkSession,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true in production (HTTPS)
	})

	http.Redirect(w, r, "/controlcenter", http.StatusFound)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     orkSession,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
