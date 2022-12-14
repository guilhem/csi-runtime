/*
Copyright 2022 Guilhem Lettron <guilhem@barpilot.io>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"errors"
	"net"
	"net/url"
	"path"
)

func ListenerFromUrl(ctx context.Context, endpoint string) (net.Listener, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		var lc net.ListenConfig

		return lc.Listen(ctx, u.Scheme, path.Join(u.Host, u.Path))
	}

	return nil, errors.New("Can't listen")
}
