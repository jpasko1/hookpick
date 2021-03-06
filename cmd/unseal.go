// Copyright © 2017 Lee Briggs <lee@leebriggs.co.uk>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/vault/api"
	v "github.com/jaxxstorm/hookpick/vault"

	"github.com/spf13/cobra"

	"sync"
)

// unsealCmd represents the unseal command
var unsealCmd = &cobra.Command{
	Use:   "unseal",
	Short: "Unseal the vaults using the key providers",
	Long: `Sends an unseal operationg to all vaults in the configuration file
using the key provided`,
	Run: func(cmd *cobra.Command, args []string) {

		datacenters := getDatacenters()
		caPath := getCaPath()
		protocol := getProtocol()

		var wg sync.WaitGroup

		for _, d := range datacenters {

			datacenter := getSpecificDatacenter()

			if datacenter == d.Name || datacenter == "" {

				var gpg bool
				var vaultKeys []string
				var gpgKey string

				for _, k := range d.Keys {
					gpg, gpgKey = getGpgKey(k.Key)
					if gpg {
						vaultKeys = append(vaultKeys, gpgKey)
					} else {
						vaultKeys = append(vaultKeys, k.Key)
					}
				}

				for _, h := range d.Hosts {
					hostName := h.Name
					hostPort := h.Port

					wg.Add(1)
					go func(hostName string, hostPort int) {
						defer wg.Done()
						client, err := v.VaultClient(hostName, hostPort, caPath, protocol)
						if err != nil {
							log.WithFields(log.Fields{"host": hostName, "port": hostPort}).Error(err)
						}

						// get the current status
						_, init := v.Status(client)
						if init {
							if len(vaultKeys) > 0 {
								var vaultStatus *api.SealStatusResponse
								for _, vaultKey := range vaultKeys {
									result, err := client.Sys().Unseal(vaultKey)
									// error while unsealing
									if err != nil {
										log.WithFields(log.Fields{"host": hostName}).Error("Error running unseal operation")
									}
									vaultStatus = result
								}
								// if it's still sealed, print the progress
								if vaultStatus.Sealed == true {
									log.WithFields(log.Fields{"host": hostName, "progress": vaultStatus.Progress, "threshold": vaultStatus.T}).Info("Unseal operation performed")
									// otherwise, tell us it's unsealed!
								} else {
									log.WithFields(log.Fields{"host": hostName, "progress": vaultStatus.Progress, "threshold": vaultStatus.T}).Info("Vault is unsealed!")
								}
							} else {
								log.WithFields(log.Fields{"host": hostName}).Error("No Key Provided")
							}
						} else {
							// sad times, not ready to be unsealed
							log.WithFields(log.Fields{"host": hostName}).Error("Vault is not ready to be unsealed")
						}
					}(hostName, hostPort)
				}
				wg.Wait()
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(unsealCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// unsealCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// unsealCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
