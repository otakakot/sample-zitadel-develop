package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func main() {
	cid := os.Getenv("CLIENT_ID")
	if cid == "" {
		cid = "" // NOTE: Run `make oidcrp` to set up.
	}

	cs := os.Getenv("CLIENT_SECRET")
	if cs == "" {
		cs = "" // NOTE: Run `make oidcrp` to set up.
	}

	iss := os.Getenv("ISSUER")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redirectURI := fmt.Sprintf("http://localhost:%s/callback", port)

	scope := []string{"openid"}

	provider, err := rp.NewRelyingPartyOIDC(
		context.Background(),
		iss,
		cid,
		cs,
		redirectURI,
		scope,
	)
	if err != nil {
		panic(err)
	}

	hdl := &Handler{
		authURI:  fmt.Sprintf("http://localhost:%s/auth", port),
		provider: provider,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", hdl.Get)
	mux.HandleFunc("/auth", hdl.Auth)
	mux.HandleFunc("/callback", hdl.Callback)

	const timeout = 30

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           mux,
		ReadHeaderTimeout: timeout * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	go func() {
		slog.Info("start server listen")

		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	<-ctx.Done()

	slog.Info("start server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}

	slog.Info("done server shutdown")
}

type Handler struct {
	authURI  string
	provider rp.RelyingParty
}

func (hdl *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	loginTmpl, _ := template.New("login").Parse(view)

	data := &struct {
		URI string
	}{
		URI: hdl.authURI,
	}

	buf := new(bytes.Buffer)

	if err := loginTmpl.Execute(buf, data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html")

	if _, err := w.Write(buf.Bytes()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

const view = `
<!DOCTYPE html>
<html>
<head>
    <title>Login</title>
    <script>
        function onLoginButtonClick() {
            navigation.navigate(encodeURI("{{.URI}}"));
        }
    </script>
</head>
<body>
    <button onclick="onLoginButtonClick()">zitadel</button>
</body>
</html>
`

func (hdl *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	state := uuid.NewString()

	endpoint := rp.AuthURL(state, hdl.provider)

	location, err := url.Parse(endpoint)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	cookie := &http.Cookie{
		Name:  "state",
		Value: state,
	}

	http.SetCookie(w, cookie)

	http.Redirect(w, r, location.String(), http.StatusFound)
}

func (hdl *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:   "state",
		Value:  "",
		MaxAge: -1,
	}

	http.SetCookie(w, cookie)

	ck, _ := r.Cookie("state")

	if ck.Value != r.URL.Query().Get("state") {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	token, err := rp.CodeExchange[*oidc.IDTokenClaims](r.Context(), r.URL.Query().Get("code"), hdl.provider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	claims, err := rp.VerifyIDToken[*oidc.IDTokenClaims](r.Context(), idToken, hdl.provider.IDTokenVerifier())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	jbonb, _ := claims.MarshalJSON()

	if _, err := w.Write(jbonb); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
