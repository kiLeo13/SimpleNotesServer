package cognitoclient

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"os"
)

var Client CognitoInterface

// User is the default user struct for all basic Cognito operations.
type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserConfirmation is the default structure for approving e-mail verification.
type UserConfirmation struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// UserLogin defines the standard structure for logging in to the application.
type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthCreate represents the response of Cognito sign in approval.
type AuthCreate struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
}

type CognitoInterface interface {
	SignUp(user *User) (string, error)
	SignIn(user *UserLogin) (*AuthCreate, error)
	GlobalSignOut(accessToken string) error
	ConfirmAccount(user *UserConfirmation) error
	ResendConfirmation(email string) error
}

type cognitoClient struct {
	cognitoClient *cognito.CognitoIdentityProvider
	appClientId   string
}

func InitCognitoClient(appClientId string) error {
	config := aws.Config{
		Region:                        aws.String(os.Getenv("AWS_COGNITO_REGION")),
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		return err
	}

	client := cognito.New(sess)
	Client = &cognitoClient{
		cognitoClient: client,
		appClientId:   appClientId,
	}
	return nil
}

// SignUp creates a new user row on Cognito and return its "sub" (the UUID)
func (c *cognitoClient) SignUp(user *User) (string, error) {
	userCognito := &cognito.SignUpInput{
		ClientId: aws.String(c.appClientId),
		Username: aws.String(user.Email),
		Password: aws.String(user.Password),
		UserAttributes: []*cognito.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(user.Email),
			},
		},
	}
	out, err := c.cognitoClient.SignUp(userCognito)
	if err != nil {
		return "", err
	}
	return *out.UserSub, nil
}

// GlobalSignOut signs out all the user session in all devices.
// In other words, it invalidates all the existing JWT tokens
func (c *cognitoClient) GlobalSignOut(accessToken string) error {
	logout := &cognito.GlobalSignOutInput{
		AccessToken: aws.String(accessToken),
	}
	_, err := c.cognitoClient.GlobalSignOut(logout)
	if err != nil {
		return err
	}
	return nil
}

// ConfirmAccount is used to verify the user's e-mail address
func (c *cognitoClient) ConfirmAccount(user *UserConfirmation) error {
	confirmationInput := &cognito.ConfirmSignUpInput{
		Username:         aws.String(user.Email),
		ConfirmationCode: aws.String(user.Code),
		ClientId:         aws.String(c.appClientId),
	}
	_, err := c.cognitoClient.ConfirmSignUp(confirmationInput)
	if err != nil {
		return err
	}
	return nil
}

// ResendConfirmation resends the verification code to the provided e-mail
func (c *cognitoClient) ResendConfirmation(email string) error {
	confirmationInput := &cognito.ResendConfirmationCodeInput{
		Username: aws.String(email),
		ClientId: aws.String(c.appClientId),
	}
	_, err := c.cognitoClient.ResendConfirmationCode(confirmationInput)
	if err != nil {
		return err
	}
	return nil
}

// SignIn signs the user in... pretty straightforward
func (c *cognitoClient) SignIn(user *UserLogin) (*AuthCreate, error) {
	authInput := &cognito.InitiateAuthInput{
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: aws.StringMap(map[string]string{
			"USERNAME": user.Email,
			"PASSWORD": user.Password,
		}),
		ClientId: aws.String(c.appClientId),
	}
	result, err := c.cognitoClient.InitiateAuth(authInput)
	if err != nil {
		return nil, err
	}
	return &AuthCreate{
		IDToken:     *result.AuthenticationResult.IdToken,
		AccessToken: *result.AuthenticationResult.AccessToken,
	}, nil
}
