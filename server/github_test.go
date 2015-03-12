package server

import (
	"fmt"
	"path"
	"testing"
)

var testClient *ReleaseManager

func TestSplitUpdateAsset(t *testing.T) {
	var err error
	var info *AssetInfo

	if info, err = getAssetInfo("autoupdate-binary-darwin-x86.dmg"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Darwin || info.Arch != Arch.X86 {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-darwin-x64.v1"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Darwin || info.Arch != Arch.X64 {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-linux-arm"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Linux || info.Arch != Arch.ARM {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-windows-x86"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Windows || info.Arch != Arch.X86 {
		t.Fatal("Failed to identify update asset.")
	}

	if _, err = getAssetInfo("autoupdate-binary-osx-x86"); err == nil {
		t.Fatalf("Should have ignored the release, \"osx\" is not a valid OS value.")
	}
}

func TestNewClient(t *testing.T) {
	testClient = NewReleaseManager("getlantern", "autoupdate-server")
	if testClient == nil {
		t.Fatal("Failed to create new client.")
	}
}

func TestListReleases(t *testing.T) {
	if _, err := testClient.GetReleases(); err != nil {
		t.Fatal(fmt.Errorf("Failed to pull releases: %q", err))
	}
}

func TestUpdateAssetsMap(t *testing.T) {
	if err := testClient.UpdateAssetsMap(); err != nil {
		t.Fatal(fmt.Errorf("Failed to update assets map: %q", err))
	}
	if testClient.updateAssetsMap == nil {
		t.Fatal("Assets map should not be nil at this point.")
	}
	if testClient.latestAssetsMap == nil {
		t.Fatal("Assets map should not be nil at this point.")
	}
}

func TestDownloadOldestVersionAndUpgradeIt(t *testing.T) {

	if len(testClient.updateAssetsMap) == 0 {
		t.Fatal("Assets map is empty.")
	}

	oldestVersionMap := make(map[string]map[string]*Asset)

	// Using the updateAssetsMap to look for the oldest version of each release.
	for os := range testClient.updateAssetsMap {
		for arch := range testClient.updateAssetsMap[os] {
			var oldestAsset *Asset
			for i := range testClient.updateAssetsMap[os][arch] {
				asset := testClient.updateAssetsMap[os][arch][i]
				if oldestAsset == nil {
					oldestAsset = asset
				} else {
					if VersionCompare(oldestAsset.v, asset.v) == Lower {
						oldestAsset = asset
					}
				}
			}
			if oldestAsset != nil {
				if oldestVersionMap[os] == nil {
					oldestVersionMap[os] = make(map[string]*Asset)
				}
				oldestVersionMap[os][arch] = oldestAsset
			}
		}
	}

	// Let's download each one of the oldest versions.
	var err error
	var p *Patch

	if len(oldestVersionMap) == 0 {
		t.Fatal("No older software versions to test with.")
	}

	tests := 0

	for os := range oldestVersionMap {
		for arch := range oldestVersionMap[os] {
			asset := oldestVersionMap[os][arch]
			newAsset := testClient.latestAssetsMap[os][arch]

			if asset == newAsset {
				t.Logf("Skipping version %s %s %s", os, arch, asset.v)
				// Skipping
				continue
			}

			// Generate a binary diff of the two assets.
			if p, err = GeneratePatch(asset.URL, newAsset.URL); err != nil {
				t.Fatal(fmt.Errorf("Unable to generate patch: %q", err))
			}

			// Apply patch.
			var oldAssetFile string
			if oldAssetFile, err = downloadAsset(asset.URL); err != nil {
				t.Fatal(err)
			}

			var newAssetFile string
			if newAssetFile, err = downloadAsset(newAsset.URL); err != nil {
				t.Fatal(err)
			}

			patchedFile := "_tests/" + path.Base(asset.URL)

			if err = bspatch(oldAssetFile, patchedFile, p.File); err != nil {
				t.Fatal(fmt.Sprintf("Failed to apply binary diff: %q", err))
			}

			// Compare the two versions.
			if fileHash(oldAssetFile) == fileHash(newAssetFile) {
				t.Fatal("Nothing to update, probably not a good test case.")
			}

			if fileHash(patchedFile) != fileHash(newAssetFile) {
				t.Fatal("File hashes after patch must be equal.")
			}

			var cs string
			if cs, err = checksumForFile(patchedFile); err != nil {
				t.Fatal("Could not get checksum for %s: %q", patchedFile, err)
			}

			if cs == asset.Checksum {
				t.Fatal("Computed checksum for patchedFile must be different than the stored older asset checksum.")
			}

			if cs != newAsset.Checksum {
				t.Fatal("Computed checksum for patchedFile must be equal to the stored newer asset checksum.")
			}

			var ss string
			if ss, err = signatureForFile(patchedFile); err != nil {
				t.Fatal("Could not get signature for %s: %q", patchedFile, err)
			}

			if ss == asset.Signature {
				t.Fatal("Computed signature for patchedFile must be different than the stored older asset signature.")
			}

			if ss != newAsset.Signature {
				t.Fatal("Computed signature for patchedFile must be equal to the stored newer asset signature.")
			}

			tests++

		}
	}

	if tests == 0 {
		t.Fatal("Seems like there is not any newer software version to test with.")
	}

}
