// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

// Package snapstate implements the manager and state aspects responsible for the installation and removal of snaps.
package snapstate

import (
	"fmt"

	"github.com/ubuntu-core/snappy/i18n"
	"github.com/ubuntu-core/snappy/osutil"
	"github.com/ubuntu-core/snappy/overlord/state"
	"github.com/ubuntu-core/snappy/snappy"
)

// Install returns a set of tasks for installing snap.
// Note that the state must be locked by the caller.
func Install(s *state.State, snap, channel string, flags snappy.InstallFlags) (*state.TaskSet, error) {
	// download
	var download *state.Task
	ss := snapSetup{
		Name:       snap,
		Channel:    channel,
		SetupFlags: int(flags),
	}
	if !osutil.FileExists(snap) {
		name, developer := snappy.SplitDeveloper(snap)
		ss.Name = name
		ss.Developer = developer
		download = s.NewTask("download-snap", fmt.Sprintf(i18n.G("Downloading %q"), snap))
	} else {
		download = s.NewTask("nop", "")
		ss.SnapPath = snap
	}
	download.Set("snap-setup", ss)

	// mount
	mount := s.NewTask("mount-snap", fmt.Sprintf(i18n.G("Mounting %q"), snap))
	mount.Set("snap-setup-task", download.ID())
	mount.WaitFor(download)

	// copy-data (needs to stop services)
	copyData := s.NewTask("copy-snap-data", fmt.Sprintf(i18n.G("Copying snap data for %q"), snap))
	copyData.Set("snap-setup-task", download.ID())
	copyData.WaitFor(mount)

	// security
	setupSecurity := s.NewTask("setup-snap-security", fmt.Sprintf(i18n.G("Setting up security profile for %q"), snap))
	setupSecurity.Set("snap-setup-task", download.ID())
	setupSecurity.WaitFor(copyData)

	// finalize (wrappers+current symlink)
	linkSnap := s.NewTask("link-snap", fmt.Sprintf(i18n.G("Final step for %q"), snap))
	linkSnap.Set("snap-setup-task", download.ID())
	linkSnap.WaitFor(setupSecurity)

	return state.NewTaskSet(download, mount, copyData, setupSecurity, linkSnap), nil
}

// Update initiates a change updating a snap.
// Note that the state must be locked by the caller.
func Update(s *state.State, snap, channel string, flags snappy.InstallFlags) (*state.TaskSet, error) {
	t := s.NewTask("update-snap", fmt.Sprintf(i18n.G("Updating %q"), snap))
	t.Set("snap-setup", snapSetup{
		Name:       snap,
		Channel:    channel,
		SetupFlags: int(flags),
	})

	return state.NewTaskSet(t), nil
}

// Remove returns a set of tasks for removing snap.
// Note that the state must be locked by the caller.
func Remove(s *state.State, snap string, flags snappy.RemoveFlags) (*state.TaskSet, error) {
	t := s.NewTask("remove-snap", fmt.Sprintf(i18n.G("Removing %q"), snap))
	t.Set("snap-setup", snapSetup{
		Name:       snap,
		SetupFlags: int(flags),
	})

	return state.NewTaskSet(t), nil
}

// Rollback returns a set of tasks for rolling back a snap.
// Note that the state must be locked by the caller.
func Rollback(s *state.State, snap, ver string) (*state.TaskSet, error) {
	t := s.NewTask("rollback-snap", fmt.Sprintf(i18n.G("Rolling back %q"), snap))
	t.Set("snap-setup", snapSetup{
		Name:    snap,
		Version: ver,
	})

	return state.NewTaskSet(t), nil
}

// Activate returns a set of tasks for activating a snap.
// Note that the state must be locked by the caller.
func Activate(s *state.State, snap string) (*state.TaskSet, error) {
	msg := fmt.Sprintf(i18n.G("Set active %q"), snap)
	t := s.NewTask("activate-snap", msg)
	t.Set("snap-setup", snapSetup{
		Name: snap,
	})

	return state.NewTaskSet(t), nil
}

// Activate returns a set of tasks for activating a snap.
// Note that the state must be locked by the caller.
func Deactivate(s *state.State, snap string) (*state.TaskSet, error) {
	msg := fmt.Sprintf(i18n.G("Set inactive %q"), snap)
	t := s.NewTask("deactivate-snap", msg)
	t.Set("snap-setup", snapSetup{
		Name: snap,
	})

	return state.NewTaskSet(t), nil
}