package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/client"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

func main() {
	slog.Info("start user creation ... ")

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

	email := fmt.Sprintf("%s@example.com", uuid.NewString())

	user, err := cli.ManagementService().AddHumanUser(ctx, &management.AddHumanUserRequest{
		UserName: uuid.NewString(),
		Profile: &management.AddHumanUserRequest_Profile{
			FirstName:   "test",
			LastName:    "test",
			NickName:    "test",
			DisplayName: "test",
		},
		Email: &management.AddHumanUserRequest_Email{
			Email:           email,
			IsEmailVerified: true,
		},
		InitialPassword: "Password1!",
	})
	if err != nil {
		panic(err)
	}

	slog.Info(fmt.Sprintf("email: %s", email))

	if _, err := cli.ManagementService().SetHumanPassword(ctx, &management.SetHumanPasswordRequest{
		UserId:           user.UserId,
		Password:         "P@ssword1",
		NoChangeRequired: true,
	}); err != nil {
		panic(err)
	}

	slog.Info("user created")
}
