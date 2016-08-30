/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"os"
	"runtime"

	l "github.com/ernestio/builder-library"
)

var s l.Scheduler

func main() {
	s.Setup(os.Getenv("NATS_URI"))

	s.ProcessRequest("instances.create", "instance.create")
	s.ProcessRequest("instances.delete", "instance.delete")
	s.ProcessRequest("instances.update", "instance.update")

	s.ProcessSuccessResponse("instance.create.done", "instance.create", "instances.create.done")
	s.ProcessSuccessResponse("instance.delete.done", "instance.delete", "instances.delete.done")
	s.ProcessSuccessResponse("instance.update.done", "instance.update", "instances.update.done")

	s.ProcessFailedResponse("instance.create.error", "instances.create.error")
	s.ProcessFailedResponse("instance.delete.error", "instances.delete.error")
	s.ProcessFailedResponse("instance.update.error", "instances.update.error")

	runtime.Goexit()
}
