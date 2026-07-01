package cookie

import (
	"net/http"

	"github.com/m-bromo/go-auth-template/configs"
)

const cookieName = "auth_cookie"

type CookieManager struct {
	cfg *configs.Config
}

func NewCookieManager(cfg *configs.Config) *CookieManager {
	return &CookieManager{
		cfg: cfg,
	}
}

func (c *CookieManager) SetCookie(w http.ResponseWriter, value string) {
	cookie := http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(c.cfg.RefreshToken.Duration.Seconds()),
		HttpOnly: true,
		Secure:   c.cfg.IsProduction(),
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
		Secure:   c.cfg.IsProduction(),
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &cookie)
}
