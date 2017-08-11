package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyExistingKeycloakUserProfileInfo(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	LastName := "lastname"
	Email := "s@s.com"
	URL := "http://noURL"

	keycloakUserProfile := &login.KeycloakUserProfile{
		LastName: &LastName,
		Email:    &Email,
		Attributes: &login.KeycloakUserProfileAttributes{
			login.URLAttributeName: {URL},
		},
	}

	Username := "user1"           // this isnt being updated
	FirstName := "firstname"      // this isnt being updated
	oldLastName := "oldlast name" // will be updated
	oldEmail := "old@email"       // will be updated
	Bio := "No more john doe"     // will  not be updated.

	existingProfile := &login.KeycloakUserProfileResponse{
		Username:  &Username,
		FirstName: &FirstName,
		LastName:  &oldLastName,
		Email:     &oldEmail,
		Attributes: &login.KeycloakUserProfileAttributes{
			login.BioAttributeName: {Bio},
			login.URLAttributeName: {URL},
		},
	}

	mergedProfile := mergeKeycloakUserProfileInfo(keycloakUserProfile, existingProfile)

	// ensure existing properties stays as is
	assert.Equal(t, *mergedProfile.Username, Username)
	assert.Equal(t, *mergedProfile.FirstName, FirstName)

	// ensure last name is updated
	assert.Equal(t, *mergedProfile.LastName, LastName)

	// ensure URL is updated to the same value

	retrievedURL := (*mergedProfile.Attributes)[login.URLAttributeName]
	require.NotEmpty(t, retrievedURL)
	assert.Equal(t, retrievedURL[0], URL)

	// ensure existing attributes dont get changed
	retrievedBio := (*mergedProfile.Attributes)[login.BioAttributeName]
	require.NotEmpty(t, retrievedBio)
	assert.Equal(t, retrievedBio[0], Bio)

}
