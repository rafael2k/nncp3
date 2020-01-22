/*
NNCP -- Node to Node copy, utilities for store-and-forward data exchange
Copyright (C) 2016-2020 Sergey Matveev <stargrave@stargrave.org>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, version 3 of the License.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package nncp

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func (ctx *Ctx) LockDir(nodeId *NodeId, lockCtx string) (*os.File, error) {
	ctx.ensureRxDir(nodeId)
	lockPath := filepath.Join(ctx.Spool, nodeId.String(), lockCtx) + ".lock"
	dirLock, err := os.OpenFile(
		lockPath,
		os.O_CREATE|os.O_WRONLY,
		os.FileMode(0666),
	)
	if err != nil {
		ctx.LogE("lockdir", SDS{"path": lockPath}, err, "")
		return nil, err
	}
	err = unix.Flock(int(dirLock.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		ctx.LogE("lockdir", SDS{"path": lockPath}, err, "")
		dirLock.Close() // #nosec G104
		return nil, err
	}
	return dirLock, nil
}

func (ctx *Ctx) UnlockDir(fd *os.File) {
	if fd != nil {
		unix.Flock(int(fd.Fd()), unix.LOCK_UN) // #nosec G104
		fd.Close() // #nosec G104
	}
}
