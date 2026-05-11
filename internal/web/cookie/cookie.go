package cookie

import "net/http"

const cookieName = "auth_cookie"

func SetCookie(w http.ResponseWriter, value string) {
	cookie := http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)
}

func GetCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	return cookie, nil
}
