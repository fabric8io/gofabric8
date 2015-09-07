/**
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"fmt"

	"github.com/daviddengcn/go-colortext"
	commands "github.com/fabric8io/gofabric8/cmds"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func main() {
	ct.ChangeColor(ct.Blue, false, ct.None, false)
	fmt.Println(fabric8AsciiArt)
	ct.ResetColor()

	cmds := &cobra.Command{
		Use:   "gofabric8",
		Short: "gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment",
		Long: `gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment
								Find more information at http://fabric8.io.`,
		Run: runHelp,
	}

	f := cmdutil.NewFactory(nil)
	f.BindFlags(cmds.PersistentFlags())

	cmds.PersistentFlags().StringP("version", "v", "latest", "fabric8 version")
	cmds.PersistentFlags().BoolP("yes", "y", false, "assume yes")

	cmds.AddCommand(commands.NewCmdValidate(f))
	cmds.AddCommand(commands.NewCmdDeploy(f))
	cmds.AddCommand(commands.NewCmdVolume(f))
	cmds.AddCommand(commands.NewCmdSecrets(f))

	cmds.Execute()
}

const fabric8AsciiArt = `
                                   Z Z        Z
                                  Z Z        Z
                                  Z Z        Z
                                  Z Z        Z
                  ZZZZZZZZZZZZ    Z Z        Z
                  Z      Z Z Z    Z Z        Z
                  Z      Z Z Z    ZZZZZZZZZZZZ
                  Z      Z Z Z                    ZZZZZZZZZZZZ
                  Z      Z Z Z                    Z Z Z      Z
                  Z      Z Z Z                    Z Z Z      Z
                  Z          Z                    Z Z Z      Z
                                                  Z Z Z      Z
                  Z                               Z Z Z      Z
                    Z                             ZZZZZZZZZZZZ
              ZZZ  ZZZ
               ZZZZZZZ
                 ZZZZZZZ                                   Z    Z
                 :  ZZZZZ          Z ZZZZZZZZ             ZZZ  ZZ
                 Z  .ZZZZZ        ZZZZZ$    ZZZ          ZZZZZZZ
                   Z  ZZZZZZZ$ZZZZZZZZZZZZZZZZZZ        ZZZZZZZ
                    ZZ  ZZZZZZZ IIIZZZZZZZZZZZZZZZ     ZZZZZZZZ
                      ZZ ,ZZZZZ, ZZZZZZZZZZ+    ZZZZ  ZZZZZZZ
                         ZZZZZZZ  ZZ:::::::::IZZZZZZZZZZZZZZ
                            ZZZZ  Z:           ,:::ZZZZZZ
                            ZZZZI Z:       I.     :?ZZ
                             ZZZZ Z:       II     :ZZ
                             ZZZZ Z:    ,IIII     :ZZ
                             ZZZZ Z:    IIII      :ZZ
                             ZZZZZZ:    II.I      :ZZ
                             ZZZZZZI:   II III   .:ZZ
                             ZZZZZZZZ::::::::::::?ZZ
                             ZZZZZZZZZZZZZZZZZZZZZZZ
                                ZZZZZZZZZZZZZZZZZZZZ
                                ZZZZZZZ     ZZZZZZ
                                ZZZZZZZ     ZZZZZZ
                                ZZZ ZZZ     ZZZZZZ
                                ZZZ ZZZZ+:::ZZZZZZZ:::::
                         :::::::::ZZZZZZZZ::::ZZZZZZ=:
                               ::::::::::::::::::::::

                  ZZZZ         ZZ             ZZ          ZZZZZZ
                  ZZ           ZZ             ZZ         ZZZ  ZZZ
                 ZZZZZ ZZZZZZ  ZZZZZZ  ZZ ZZZ ZZ  ZZZZZZ ZZ   ZZZ
                  ZZ       ZZZ ZZ   ZZ ZZZZZZ ZZ ZZZ     ZZZZZZZ
                  ZZ   ZZZZZZZ ZZ   ZZ ZZZ    ZZ ZZ      ZZ   ZZZ
                  ZZ   ZZ  ZZZ ZZ   ZZ ZZZ    ZZ ZZ      ZZ   ZZZ
                  ZZ   ZZZZZZZ ZZZZZZZ ZZZ    ZZ ZZZZZZZ ZZZZZZZZ
`
