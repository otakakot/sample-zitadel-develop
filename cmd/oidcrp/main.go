package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/client"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/app"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
	"google.golang.org/protobuf/types/known/durationpb"
)

func main() {
	slog.Info("start oidc rp creation ... ")

	ctx := context.Background()

	initial := client.DefaultServiceUserAuthentication("./machinekey/zitadel-admin-sa.json", oidc.ScopeOpenID, client.ScopeZitadelAPI())

	cli, err := client.New(
		ctx,
		zitadel.New("localhost", zitadel.WithInsecure("8080")),
		client.WithAuth(initial),
	)
	if err != nil {
		panic(err)
	}

	projects, err := cli.ManagementService().ListProjects(ctx, &management.ListProjectsRequest{})
	if err != nil {
		panic(err)
	}

	projectID := projects.Result[0].Id

	app, err := cli.ManagementService().AddOIDCApp(ctx, &management.AddOIDCAppRequest{
		ProjectId:                projectID,
		Name:                     uuid.NewString(),
		RedirectUris:             []string{"http://localhost:7070/callback"},
		ResponseTypes:            []app.OIDCResponseType{app.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		GrantTypes:               []app.OIDCGrantType{app.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE},
		AppType:                  app.OIDCAppType_OIDC_APP_TYPE_WEB,
		AuthMethodType:           app.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
		PostLogoutRedirectUris:   []string{},
		Version:                  0,
		DevMode:                  true,
		AccessTokenType:          app.OIDCTokenType_OIDC_TOKEN_TYPE_JWT,
		AccessTokenRoleAssertion: false,
		IdTokenRoleAssertion:     false,
		IdTokenUserinfoAssertion: false,
		ClockSkew:                &durationpb.Duration{},
		AdditionalOrigins:        []string{},
		SkipNativeAppSuccessPage: false,
	})
	if err != nil {
		panic(err)
	}

	slog.Info(fmt.Sprintf("client id: %s", app.ClientId))
	slog.Info(fmt.Sprintf("client secret: %s", app.ClientSecret))

	slog.Info("oidc rp created")
}
