package gmail

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"math/rand"

	"net"
	"net/http"

	"encoding/json"

	"github.com/pkg/browser"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	googleOauth2Client = "347154790595-fd0ar8ngcq2qbmafdju3safg32fkcpnk.apps.googleusercontent.com"
	googleOauth2Secret = "aSPhfsqxCRSr1cU8RNWZ-r9S"
	oauth2Endpoint     = "/oauth2/"
)

func createResponseListener() (*http.Server, *chan string) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	c := make(chan string, 1)
	if err != nil {
		panic(err)
	}
	m := http.NewServeMux()
	srv := &http.Server{Handler: m, Addr: l.Addr().String()}
	m.HandleFunc(oauth2Endpoint, func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("You can now close the browser"))
		r := req.URL.Query().Get("code")
		if r != "" {
			c <- r
		}
	})
	go func() {
		srv.Serve(l)
		close(c)
	}()
	return srv, &c
}

func randomBytes() *[]byte {
	rb := make([]byte, 64)
	if _, err := rand.Read(rb); err != nil {
		panic(err)
	}
	return &rb
}

func makeChallenge(v *string) string {
	s256 := sha256.Sum256([]byte(*v))
	return base64.RawURLEncoding.EncodeToString(s256[:])
}

func storeToken(t *oauth2.Token, inMemory bool) {
	tb, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	if err := keyring.Set("orion-core", "user-token", string(tb)); err != nil {
		panic(err)
	}
}

// New implements OAuth2 installed application flow and returns a gmail service client.
func New() *gmail.Service {
	srv, c := createResponseListener()
	defer srv.Shutdown(context.Background())
	config := &oauth2.Config{
		ClientID:     googleOauth2Client,
		ClientSecret: googleOauth2Secret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://" + srv.Addr + oauth2Endpoint,
		Scopes:       []string{gmail.MailGoogleComScope},
	}
	verifier := base64.RawURLEncoding.EncodeToString(*randomBytes())
	challenge := makeChallenge(&verifier)
	url := config.AuthCodeURL(
		"state",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("response_type", "code"),
	)
	browser.OpenURL(url)
	code := <-*c
	token, err := config.Exchange(
		context.Background(),
		code,
		oauth2.SetAuthURLParam("grant_type", "authorization_code"),
		oauth2.SetAuthURLParam("code_verifier", verifier),
		oauth2.SetAuthURLParam("client_id", googleOauth2Client),
		oauth2.SetAuthURLParam("redirect_uri", "http://"+srv.Addr+oauth2Endpoint),
	)
	if err != nil {
		panic(err)
	}
	ts := config.TokenSource(context.Background(), token)
	client, err := gmail.NewService(
		context.Background(),
		option.WithTokenSource(ts),
	)
	if err != nil {
		panic(err)
	}
	return client
}
