package plg_authenticate_forwardauth

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"

	. "github.com/mickael-kerjean/filestash/server/common"
)

const (
	defaultUsernameHeader = "Remote-User"
	defaultEmailHeader    = "Remote-Email"
	defaultPasswordHeader = "Remote-Token"
	backendLabel          = "default"
)

func init() {
	Hooks.Register.AuthenticationMiddleware("forwardauth", ForwardAuth{})
	Hooks.Register.Onload(autoconfigure)
}

func autoconfigure() {
	webdavURL := os.Getenv("WEBDAV_URL")
	if webdavURL == "" {
		return
	}
	usernameHeader := envOr("FORWARD_AUTH_USERNAME_HEADER", defaultUsernameHeader)
	emailHeader := envOr("FORWARD_AUTH_EMAIL_HEADER", defaultEmailHeader)
	passwordHeader := envOr("FORWARD_AUTH_PASSWORD_HEADER", defaultPasswordHeader)

	Config.Get("middleware.identity_provider.type").Set("forwardauth")
	idpParams, _ := json.Marshal(map[string]string{
		"type":            "forwardauth",
		"username_header": usernameHeader,
		"email_header":    emailHeader,
		"password_header": passwordHeader,
	})
	Config.Get("middleware.identity_provider.params").Set(string(idpParams))

	webdavConn := map[string]string{
		"type":     "webdav",
		"url":      webdavURL,
		"username": "{{ .user }}",
		"password": "{{ .password }}",
	}
	if path := os.Getenv("WEBDAV_PATH"); path != "" {
		webdavConn["path"] = path
	}
	mapping, _ := json.Marshal(map[string]map[string]string{backendLabel: webdavConn})
	Config.Get("middleware.attribute_mapping.related_backend").Set(backendLabel)
	Config.Get("middleware.attribute_mapping.params").Set(string(mapping))

	Config.Conn = []map[string]any{
		{"type": "webdav", "label": backendLabel},
	}

	if Config.Get("auth.admin").String() == "" {
		Config.Get("auth.admin").Set("disabled-forwardauth")
	}

	Config.Save()
	fmt.Fprintln(os.Stderr, "[forwardauth] autoconfigured webdav backend at", webdavURL)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type ForwardAuth struct{}

func (this ForwardAuth) Setup() Form {
	return Form{
		Elmnts: []FormElement{
			{
				Name:  "type",
				Type:  "hidden",
				Value: "forwardauth",
			},
			{
				Name:        "username_header",
				Type:        "text",
				Value:       defaultUsernameHeader,
				Placeholder: defaultUsernameHeader,
				Description: "Request header carrying the authenticated username (exposed as {{ .user }}).",
			},
			{
				Name:        "email_header",
				Type:        "text",
				Value:       defaultEmailHeader,
				Placeholder: defaultEmailHeader,
				Description: "Request header carrying the user's email (exposed as {{ .email }}).",
			},
			{
				Name:        "password_header",
				Type:        "text",
				Value:       defaultPasswordHeader,
				Placeholder: defaultPasswordHeader,
				Description: "Request header carrying the user's per-user WebDAV API key (exposed as {{ .password }}).",
			},
		},
	}
}

func headerOrDefault(idpParams map[string]string, key, def string) string {
	if v := idpParams[key]; v != "" {
		return v
	}
	return def
}

func (this ForwardAuth) EntryPoint(idpParams map[string]string, req *http.Request, res http.ResponseWriter) error {
	usernameHeader := headerOrDefault(idpParams, "username_header", defaultUsernameHeader)
	emailHeader := headerOrDefault(idpParams, "email_header", defaultEmailHeader)
	passwordHeader := headerOrDefault(idpParams, "password_header", defaultPasswordHeader)

	user := req.Header.Get(usernameHeader)
	email := req.Header.Get(emailHeader)
	password := req.Header.Get(passwordHeader)

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	if user == "" || password == "" {
		res.WriteHeader(http.StatusUnauthorized)
		res.Write([]byte(Page(`
            <h1>Forward-auth headers missing</h1>
            <p>
                Filestash is configured to read identity from forward-auth headers
                (<code>` + html.EscapeString(usernameHeader) + `</code>,
                <code>` + html.EscapeString(passwordHeader) + `</code>) but at least one was not present
                on this request. This instance must sit behind a forward-auth proxy that injects
                those headers.
            </p>
        `)))
		return nil
	}

	getParams := "?label=" + html.EscapeString(req.URL.Query().Get("label")) +
		"&state=" + html.EscapeString(req.URL.Query().Get("state"))

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(Page(`
        <form action="` + WithBase("/api/session/auth/"+getParams) + `" method="post">
            <input type="hidden" name="user" value="` + html.EscapeString(user) + `" />
            <input type="hidden" name="email" value="` + html.EscapeString(email) + `" />
            <input type="hidden" name="password" value="` + html.EscapeString(password) + `" />
        </form>
        <script>document.querySelector("form").submit();</script>
    `)))
	return nil
}

func (this ForwardAuth) Callback(formData map[string]string, idpParams map[string]string, res http.ResponseWriter) (map[string]string, error) {
	user := formData["user"]
	password := formData["password"]
	if user == "" || password == "" {
		return nil, ErrAuthenticationFailed
	}
	return map[string]string{
		"user":     user,
		"email":    formData["email"],
		"password": password,
	}, nil
}
