// Update protocol:
//
//   GET hk.heroku.com/hk/linux-amd64.json
//
//   200 ok
//   {
//       "Version": "2",
//       "Sha256": "..." // base64
//   }
//
// then
//
//   GET hkpatch.s3.amazonaws.com/hk/1/2/linux-amd64
//
//   200 ok
//   [bsdiff data]
//
// or
//
//   GET hkdist.s3.amazonaws.com/hk/2/linux-amd64.gz
//
//   200 ok
//   [gzipped executable data]
//
//
package selfupdate

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	update "github.com/inconshreveable/go-update"
	"github.com/kr/binarydist"
)

const (
	upcktimePath = "cktime"
	plat         = runtime.GOOS + "-" + runtime.GOARCH
)

const devValidTime = 7 * 24 * time.Hour

var ErrHashMismatch = errors.New("new file hash mismatch after patch")
var up = update.New()

// Updater is the configuration and runtime data for doing an update.
//
// Note that ApiURL, BinURL and DiffURL should have the same value if all files are available at the same location.
//
// Example:
//
//  updater := &selfupdate.Updater{
//  	CurrentVersion: version,
//  	ApiURL:         "http://updates.yourdomain.com/",
//  	BinURL:         "http://updates.yourdownmain.com/",
//  	DiffURL:        "http://updates.yourdomain.com/",
//  	Dir:            "update/",
//  	CmdName:        "myapp", // app name
//  }
//  if updater != nil {
//  	go updater.BackgroundRun()
//  }
type Updater struct {
	CurrentVersion string // Currently running version.
	ApiURL         string // Base URL for API requests (json files).
	CmdName        string // Command name is appended to the ApiURL like http://apiurl/CmdName/. This represents one binary.
	BinURL         string // Base URL for full binary downloads.
	DiffURL        string // Base URL for diff downloads.
	Dir            string // Directory to store selfupdate state.
	Info           struct {
		Version string
		Sha256  []byte
	}
}

func (u *Updater) getExecRelativeDir(dir string) string {
	if dir[0] == '/' {
		return dir
	}
	filename, _ := os.Executable()
	path := filepath.Join(filepath.Dir(filename), dir)
	return path
}

// BackgroundRun starts the update check and apply cycle.
func (u *Updater) BackgroundRun() error {
	os.MkdirAll(u.getExecRelativeDir(u.Dir), 0777)
	if err := up.CanUpdate(); err != nil {
		// fail
		return err
	}
	//self, err := osext.Executable()
	//if err != nil {
	// fail update, couldn't figure out path to self
	//return
	//}
	// TODO(bgentry): logger isn't on Windows. Replace w/ proper error reports.
	if err := u.update(); err != nil {
		return err
	}
	return nil
}

func (u *Updater) update() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	old, err := os.Open(path)
	if err != nil {
		return err
	}
	defer old.Close()

	err = u.FetchInfo()
	if err != nil {
		return err
	}
	if u.Info.Version == u.CurrentVersion {
		log.Println("update: no new version available")
		return nil
	}
	log.Println("update: fetching binary patch")
	bin, err := u.fetchAndVerifyPatch(old)
	if err != nil {
		if err == ErrHashMismatch {
			log.Println("update: hash mismatch from patched binary")
		} else {
			if u.DiffURL != "" {
				log.Println("update: error while patching binary,", err)
			}
		}

		log.Println("update: fetching full binary")
		bin, err = u.fetchAndVerifyFullBin()
		if err != nil {
			if err == ErrHashMismatch {
				log.Println("update: hash mismatch from full binary")
			} else {
				log.Println("update: error while fetching full binary,", err)
			}
			return err
		}
	}

	// close the old binary before installing because on windows
	// it can't be renamed if a handle to the file is still open
	old.Close()

	err, errRecover := up.FromStream(bytes.NewBuffer(bin))
	if errRecover != nil {
		return fmt.Errorf("update and recovery errors: %q %q", err, errRecover)
	}
	if err != nil {
		return err
	}
	log.Println("update: software updated correctly")
	return nil
}

// FetchInfo downloads info about latest available version
func (u *Updater) FetchInfo() error {
	r, err := fetch(u.ApiURL + u.CmdName + "/" + plat + ".json")
	if err != nil {
		return err
	}
	defer r.Close()
	err = json.NewDecoder(r).Decode(&u.Info)
	if err != nil {
		return err
	}
	if len(u.Info.Sha256) != sha256.Size {
		return errors.New("bad cmd hash in info")
	}
	return nil
}

func (u *Updater) fetchAndVerifyPatch(old io.Reader) ([]byte, error) {
	bin, err := u.fetchAndApplyPatch(old)
	if err != nil {
		return nil, err
	}
	if !verifySha(bin, u.Info.Sha256) {
		return nil, ErrHashMismatch
	}
	return bin, nil
}

func (u *Updater) fetchAndApplyPatch(old io.Reader) ([]byte, error) {
	r, err := fetch(u.DiffURL + u.CmdName + "/" + u.CurrentVersion + "/" + u.Info.Version + "/" + plat)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	err = binarydist.Patch(old, &buf, r)
	return buf.Bytes(), err
}

func (u *Updater) fetchAndVerifyFullBin() ([]byte, error) {
	bin, err := u.fetchBin()
	if err != nil {
		return nil, err
	}
	verified := verifySha(bin, u.Info.Sha256)
	if !verified {
		return nil, ErrHashMismatch
	}
	return bin, nil
}

func (u *Updater) fetchBin() ([]byte, error) {
	r, err := fetch(u.BinURL + u.CmdName + "/" + u.Info.Version + "/" + plat + ".gz")
	if err != nil {
		return nil, err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(buf, gz); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func fetch(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad http status from %s: %v", url, resp.Status)
	}
	return resp.Body, nil
}

func verifySha(bin []byte, sha []byte) bool {
	h := sha256.New()
	h.Write(bin)
	return bytes.Equal(h.Sum(nil), sha)
}
