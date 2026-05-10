package plg_authenticate_forwardauth

import (
	"html"
	"net/http"

	. "github.com/mickael-kerjean/filestash/server/common"
)

const (
	defaultUsernameHeader = "Remote-User"
	defaultEmailHeader    = "Remote-Email"
	defaultPasswordHeader = "Remote-Token"
)

func init() {
	Hooks.Register.AuthenticationMiddleware("forwardauth", ForwardAuth{})
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
