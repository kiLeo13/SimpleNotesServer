package cognitoclient

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// User is the default user struct for all basic Cognito operations.
type User struct {
	Email    string
	Password string
}

// UserConfirmation is the default structure for approving e-mail verification.
type UserConfirmation struct {
	Email string
	Code  string
}

// UserLogin defines the standard structure for logging in to the application.
type UserLogin struct {
	Email    string
	Password string
}

// AuthCreate represents the response of Cognito sign in approval.
type AuthCreate struct {
	IDToken     string
	AccessToken string
}

type CognitoInterface interface {
	//==============================//
	//                              //
	//     Self-user Operations     //
	//                              //
	//==============================//

	// SignUp creates a new user row on Cognito and return its "sub" (the UUID).
	SignUp(user *User) (string, error)

	// SignIn signs the user in and returns its respective access and ID tokens.
	SignIn(user *UserLogin) (*AuthCreate, error)

	// GlobalSignOut signs out all the user session in all devices.
	// In other words, it invalidates all the existing JWT tokens.
	GlobalSignOut(accessToken string) error

	// ConfirmAccount is used to verify the user's e-mail address.
	ConfirmAccount(user *UserConfirmation) error

	// ResendConfirmation resends the verification code to the provided e-mail.
	ResendConfirmation(email string) error

	//==========================//
	//                          //
	//     Admin Operations     //
	//                          //
	//==========================//

	// AdminDeleteUser deletes a user by their email on behalf of the application.
	AdminDeleteUser(email string) error
}

type cognitoClient struct {
	cognitoClient *cognitoidentityprovider.Client
	poolId        string
	appClientId   string
}

func InitCognitoClient() (CognitoInterface, error) {
	appClientId := os.Getenv("AWS_COGNITO_CLIENT_ID")
	region := os.Getenv("AWS_COGNITO_REGION")
	poolId := os.Getenv("AWS_COGNITO_USER_POOL_ID")
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := cognitoidentityprovider.NewFromConfig(cfg)
	return &cognitoClient{
		cognitoClient: client,
		poolId:        poolId,
		appClientId:   appClientId,
	}, nil
}

func (c *cognitoClient) SignUp(user *User) (string, error) {
	input := &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(c.appClientId),
		Username: aws.String(user.Email),
		Password: aws.String(user.Password),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(user.Email),
			},
		},
	}
	out, err := c.cognitoClient.SignUp(context.Background(), input)
	if err != nil {
		return "", err
	}
	return aws.ToString(out.UserSub), nil
}

func (c *cognitoClient) GlobalSignOut(accessToken string) error {
	input := &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(accessToken),
	}
	_, err := c.cognitoClient.GlobalSignOut(context.Background(), input)
	return err
}

func (c *cognitoClient) ConfirmAccount(user *UserConfirmation) error {
	input := &cognitoidentityprovider.ConfirmSignUpInput{
		Username:         aws.String(user.Email),
		ConfirmationCode: aws.String(user.Code),
		ClientId:         aws.String(c.appClientId),
	}
	_, err := c.cognitoClient.ConfirmSignUp(context.Background(), input)
	return err
}

func (c *cognitoClient) ResendConfirmation(email string) error {
	input := &cognitoidentityprovider.ResendConfirmationCodeInput{
		Username: aws.String(email),
		ClientId: aws.String(c.appClientId),
	}
	_, err := c.cognitoClient.ResendConfirmationCode(context.Background(), input)
	return err
}

func (c *cognitoClient) SignIn(user *UserLogin) (*AuthCreate, error) {
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		AuthParameters: map[string]string{
			"USERNAME": user.Email,
			"PASSWORD": user.Password,
		},
		ClientId: aws.String(c.appClientId),
	}
	result, err := c.cognitoClient.InitiateAuth(context.Background(), input)
	if err != nil {
		return nil, err
	}
	return &AuthCreate{
		IDToken:     *result.AuthenticationResult.IdToken,
		AccessToken: *result.AuthenticationResult.AccessToken,
	}, nil
}

func (c *cognitoClient) AdminDeleteUser(email string) error {
	input := &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(c.poolId),
		Username:   aws.String(email),
	}
	_, err := c.cognitoClient.AdminDeleteUser(context.Background(), input)
	return err
}
