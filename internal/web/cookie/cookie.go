package cookie

import (
	"net/http"

	"github.com/m-bromo/go-auth-template/configs"
)

const cookieName = "auth_cookie"

type CookieManager struct {
	environmentOptions  configs.Environment
	refreshTokenOptions *configs.RefreshToken
}

func NewCookieManager(
	environmentOptions configs.Environment,
	refreshTokenOptions *configs.RefreshToken,
) *CookieManager {
	return &CookieManager{
		environmentOptions:  environmentOptions,
		refreshTokenOptions: refreshTokenOptions,
	}
}

func (c *CookieManager) SetCookie(w http.ResponseWriter, value string) {
	cookie := http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(c.refreshTokenOptions.Duration.Seconds()),
		HttpOnly: true,
		Secure:   c.environmentOptions == configs.Production,
		SameSite: http.SameSiteDefaultMode,
	}

	http.SetCookie(w, &cookie)
}

func (c *CookieManager) GetCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	return cookie, nil
}

func (c *CookieManager) DeleteCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.environmentOptions == configs.Production,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &cookie)
}
