package goautowp

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autowp/goautowp/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUploadPictureTooSmall(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	brandID := createItem(t, conn, cnt, &APIItem{
		Name:       fmt.Sprintf("brand-%d", random.Int()),
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_BRAND,
		Catname:    fmt.Sprintf("brand-%d", random.Int()),
	})

	picturesREST, err := cnt.PicturesREST()
	require.NoError(t, err)

	router := gin.New()
	kc := cnt.Keycloak()
	ctx := t.Context()
	cfg := config.LoadConfig(".")

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	picturesREST.SetupRouter(router)

	req := CreatePictureRequest(t, "./test/10x10.png", PicturePostForm{ItemID: brandID}, adminToken.AccessToken)

	resRecorder := httptest.NewRecorder()
	router.ServeHTTP(resRecorder, req)

	body, err := io.ReadAll(resRecorder.Result().Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "640x360")
	require.Equal(t, http.StatusBadRequest, resRecorder.Code)
}

func TestUploadPicture(t *testing.T) {
	t.Parallel()

	random := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	brandID := createItem(t, conn, cnt, &APIItem{
		Name:       fmt.Sprintf("brand-%d", random.Int()),
		IsGroup:    true,
		ItemTypeId: ItemType_ITEM_TYPE_BRAND,
		Catname:    fmt.Sprintf("brand-%d", random.Int()),
	})

	picturesREST, err := cnt.PicturesREST()
	require.NoError(t, err)

	router := gin.New()
	kc := cnt.Keycloak()
	ctx := t.Context()
	cfg := config.LoadConfig(".")

	// admin
	adminToken, err := kc.Login(ctx, "frontend", "", cfg.Keycloak.Realm, adminUsername, adminPassword)
	require.NoError(t, err)
	require.NotNil(t, adminToken)

	picturesREST.SetupRouter(router)

	req := CreatePictureRequest(t, "./test/test.jpg", PicturePostForm{ItemID: brandID}, adminToken.AccessToken)

	resRecorder := httptest.NewRecorder()
	router.ServeHTTP(resRecorder, req)

	body, err := io.ReadAll(resRecorder.Result().Body)
	require.NoError(t, err)

	st := struct {
		ID string `json:"id"`
	}{}

	err = json.Unmarshal(body, &st)
	require.NoError(t, err, "failed to decode json. `%s` given", string(body))

	require.NotEmpty(t, st.ID, "json not contains picture.id. `%s` given", string(body))
	require.Equal(t, http.StatusCreated, resRecorder.Code)
}
