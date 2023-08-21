// Copyright 2021 Couchbase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh

import (
	"bytes"
	"os"
	"strings"

	"github.com/jamesl33/cbtools-autobench/value"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// trimPort returns the given host with the port trimmed. If the provided host does not contain a port, the string will
// be returned unchanged.
func trimPort(s string) string {
	if index := strings.Index(s, ":"); index != 0 {
		return s[:index]
	}

	return s
}

// parsePrivateKey returns a signer which can be used to authenticate ssh connections. If a passphrase is provided, the
// private key will be decrypted.
func parsePrivateKey(path, passphrase string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file at '%s'", path)
	}

	if passphrase == "" {
		return ssh.ParsePrivateKey(data)
	}

	return ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
}

// executeCommand will execute the given command using the provided client and returns the combined output.
func executeCommand(client *ssh.Client, command string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create session")
	}
	defer session.Close()

	fields := log.Fields{"remote": trimPort(client.RemoteAddr().String()), "command": command}
	log.WithFields(fields).Debug("Executing remote command")

	output, err := session.CombinedOutput(command)
	if err == nil {
		return output, nil
	}

	if len(strings.TrimSpace(string(output))) != 0 {
		log.Errorf("%s", output)
	}

	return nil, err
}

// determinePlatform uses the provided ssh client to determine which platform it's connected too.
func determinePlatform(client *ssh.Client) (value.Platform, error) {
	command := value.NewCommand("cat /etc/os-release | grep '^ID=' | cut -c4-")

	distro, err := executeCommand(client, command.ToString(nil))
	if err != nil {
		return "", errors.Wrap(err, "failed to determine distribution")
	}

	command = value.NewCommand("cat /etc/os-release | grep '^VERSION_ID=' | cut -c13- | rev | cut -c2- | rev")

	release, err := executeCommand(client, command.ToString(nil))
	if err != nil {
		return "", errors.Wrap(err, "failed to determine version")
	}

	// Do some cleanup since we don't always get uniform output
	distro = bytes.TrimSpace(distro)
	distro = bytes.TrimPrefix(distro, []byte{'"'})
	distro = bytes.TrimSuffix(distro, []byte{'"'})

	switch string(distro) {
	case "ubuntu":
		return determineUbuntuPlatform(strings.TrimSpace(string(release)))
	case "amzn":
		return determineAmazonLinuxPlatform(strings.TrimSpace(string(release)))
	}

	return "", errors.Errorf("unsupported distro '%s'", strings.TrimSpace(string(distro)))
}

// determineUbuntuPlatform returns the specific platform for the given Ubuntu release.
func determineUbuntuPlatform(release string) (value.Platform, error) {
	switch release {
	case "20.04":
		return value.PlatformUbuntu20_04, nil
	}

	return "", errors.Errorf("unsupported ubuntu release '%s'", release)
}

// determineAmazonLinuxPlatform returns the specific platform for the given Amazon Linux release.
func determineAmazonLinuxPlatform(release string) (value.Platform, error) {
	switch release {
	case "2":
		return value.PlatformAmazonLinux2, nil
	}

	return "", errors.Errorf("unsupported amazon linux release '%s'", release)
}
