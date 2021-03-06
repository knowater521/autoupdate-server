package server

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/getlantern/go-update"
	"github.com/getlantern/go-update/check"
)

const (
	localAddr  = "127.0.0.1:1123"
	publicAddr = localAddr
)

func init() {
	SetPrivateKey("../_resources/example-keys/private.key")
}

func TestStartServer(t *testing.T) {
	releaseManager := NewReleaseManager("getlantern", "lantern")
	if err := releaseManager.UpdateAssetsMap(); err != nil {
		t.Fatalf("UpdateAssetsMap: %v", err)
	}

	updateServer := &UpdateServer{
		ReleaseManager: releaseManager,
		PublicAddr:     publicAddr,
		LocalAddr:      localAddr,
	}

	go func(t *testing.T) {
		if err := updateServer.ListenAndServe(); err != nil {
			t.Fatalf("ListenAndServe: %v", err)
		}
	}(t)

	time.Sleep(time.Second * 1)
}

func TestReachServer(t *testing.T) {
	var up *update.Update

	publicKey, err := ioutil.ReadFile("../_resources/example-keys/public.pub")
	if err != nil {
		t.Fatalf("Failed to open public key: %v", err)
	}

	param := check.Params{
		AppVersion: "3.7.1",
	}

	up = update.New().ApplyPatch(update.PATCHTYPE_BSDIFF)

	if _, err = up.VerifySignatureWithPEM(publicKey); err != nil {
		t.Fatal("VerifySignatureWithPEM", err)
	}

	res, err := param.CheckForUpdate(fmt.Sprintf("http://%s/update", localAddr), up)
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}

	if res.Url == "" {
		t.Fatal("Expecting some URL.")
	}
}
