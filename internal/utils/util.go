// This file is part of arduino aws-s3-integration.
//
// Copyright 2024 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the Mozilla Public License Version 2.0,
// which covers the main part of aws-s3-integration.
// The terms of this license can be found at:
// https://www.mozilla.org/media/MPL/2.0/index.815ca599c9df.txt
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package utils

import "strings"

func StringPointer(val string) *string {
	return &val
}

func BoolPointer(val bool) *bool {
	return &val
}

func ParseTags(tags *string) map[string]string {
	tagsMap := make(map[string]string)
	if tags == nil || *tags == "" {
		println("No tags")
		return tagsMap
	}
	tagsList := strings.Split(*tags, ",")
	for _, tag := range tagsList {
		parts := strings.Split(tag, "=")
		if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
			tagsMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return tagsMap
}
