/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import "runtime"

func main() {
	n := natsClient()
	r := redisClient()

	// Process requests
	processRequest(n, r, "instances.delete", "instance.delete")
	processRequest(n, r, "instances.create", "instance.create")
	processRequest(n, r, "instances.update", "instance.update")

	// Process resulting success
	processResponse(n, r, "instance.create.done", "instances.create.", "instance.create", "completed")
	processResponse(n, r, "instance.delete.done", "instances.delete.", "instance.delete", "completed")
	processResponse(n, r, "instance.update.done", "instances.update.", "instance.update", "completed")

	// Process resulting errors
	processResponse(n, r, "instance.delete.error", "instances.delete.", "instance.delete", "errored")
	processResponse(n, r, "instance.create.error", "instances.create.", "instance.create", "errored")
	processResponse(n, r, "instance.update.error", "instances.update.", "instance.update", "errored")

	runtime.Goexit()
}
