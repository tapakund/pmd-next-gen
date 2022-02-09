// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 VMware, Inc.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/pmd-nextgen/pkg/validator"
	"github.com/pmd-nextgen/pkg/web"
	"github.com/pmd-nextgen/plugins/tdnf"
)

type ItemListDesc struct {
	Success bool            `json:"success"`
	Message []tdnf.ListItem `json:"message"`
	Errors  string          `json:"errors"`
}

type RepoListDesc struct {
	Success bool        `json:"success"`
	Message []tdnf.Repo `json:"message"`
	Errors  string      `json:"errors"`
}

type InfoListDesc struct {
	Success bool        `json:"success"`
	Message []tdnf.Info `json:"message"`
	Errors  string      `json:"errors"`
}

type NilDesc struct {
	Success bool   `json:"success"`
	Errors  string `json:"errors"`
}

type StatusDesc struct {
	Success bool               `json:"success"`
	Message web.StatusResponse `json:"message"`
	Errors  string             `json:"errors"`
}

func tdnfParseFlags(c *cli.Context) tdnf.Options {
	var options tdnf.Options

	v := reflect.ValueOf(&options).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		name := strings.ToLower(field.Name)
		value := v.Field(i).Interface()
		switch value.(type) {
		case bool:
			v.Field(i).SetBool(c.Bool(name))
		case string:
			v.Field(i).SetString(c.String(name))
		case []string:
			str := c.String(name)
			if str != "" {
				list := strings.Split(str, ",")
				size := len(list)
				if size > 0 {
					v.Field(i).Set(reflect.MakeSlice(reflect.TypeOf([]string{}), size, size))
					for j, s := range list {
						v.Field(i).Index(j).Set(reflect.ValueOf(s))
					}
				}
			}
		}
	}
	return options
}

func tdnfCreateFlags() []cli.Flag {
	var options tdnf.Options
	var flags []cli.Flag

	v := reflect.ValueOf(&options).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		name := strings.ToLower(field.Name)
		value := v.Field(i).Interface()
		switch value.(type) {
		case bool:
			flags = append(flags, &cli.BoolFlag{Name: name})
		case string:
			flags = append(flags, &cli.StringFlag{Name: name})
		case []string:
			flags = append(flags, &cli.StringFlag{Name: name, Usage: "Separate by ,"})
		}
	}
	return flags
}

func tdnfOptionsQuery(options *tdnf.Options) string {
	var qlist []string

	v := reflect.ValueOf(options).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		name := strings.ToLower(field.Name)
		value := v.Field(i).Interface()
		switch value.(type) {
		case bool:
			if value.(bool) {
				qlist = append(qlist, name+"=true")
			}
		case string:
			str := value.(string)
			if str != "" {
				qlist = append(qlist, name+"="+str)
			}
		case []string:
			list := value.([]string)
			if len(list) != 0 {
				for _, s := range list {
					qlist = append(qlist, name+"="+s)
				}
			}
		}
	}

	var qstr string
	for i, s := range qlist {
		sep := "&"
		if i == 0 {
			sep = "?"
		}
		qstr = qstr + sep + s
	}
	return qstr
}

func displayTdnfList(l *ItemListDesc) {
	for _, i := range l.Message {
		fmt.Printf("%v %v\n", color.HiBlueString("Name:"), i.Name)
		fmt.Printf("%v %v\n", color.HiBlueString("Arch:"), i.Arch)
		fmt.Printf("%v %v\n", color.HiBlueString(" Evr:"), i.Evr)
		fmt.Printf("%v %v\n", color.HiBlueString("Repo:"), i.Repo)
		fmt.Printf("\n")
	}
}

func displayTdnfRepoList(l *RepoListDesc) {
	for _, r := range l.Message {
		fmt.Printf("%v %v\n", color.HiBlueString("   Repo:"), r.Repo)
		fmt.Printf("%v %v\n", color.HiBlueString("   Name:"), r.RepoName)
		fmt.Printf("%v %v\n", color.HiBlueString("Enabled:"), r.Enabled)
		fmt.Printf("\n")
	}
}

func displayTdnfInfoList(l *InfoListDesc) {
	for _, i := range l.Message {
		fmt.Printf("%v %v\n", color.HiBlueString("        Name:"), i.Name)
		fmt.Printf("%v %v\n", color.HiBlueString("        Arch:"), i.Arch)
		fmt.Printf("%v %v\n", color.HiBlueString("         Evr:"), i.Evr)
		fmt.Printf("%v %v\n", color.HiBlueString("Install Size:"), i.InstallSize)
		fmt.Printf("%v %v\n", color.HiBlueString("        Repo:"), i.Repo)
		fmt.Printf("%v %v\n", color.HiBlueString("     Summary:"), i.Summary)
		fmt.Printf("%v %v\n", color.HiBlueString("         Url:"), i.Url)
		fmt.Printf("%v %v\n", color.HiBlueString("     License:"), i.License)
		fmt.Printf("%v %v\n", color.HiBlueString(" Description:"), i.Description)
		fmt.Printf("\n")
	}
}

func acquireTdnfList(options *tdnf.Options, pkg string, host string, token map[string]string) (*ItemListDesc, error) {
	var path string
	if !validator.IsEmpty(pkg) {
		path = "/api/v1/tdnf/list/" + pkg
	} else {
		path = "/api/v1/tdnf/list"
	}
	path = path + tdnfOptionsQuery(options)

	resp, err := web.DispatchAndWait(http.MethodGet, host, path, token, nil)
	if err != nil {
		return nil, err
	}

	m := ItemListDesc{}
	if err := json.Unmarshal(resp, &m); err != nil {
		os.Exit(1)
	}

	if m.Success {
		return &m, nil
	}

	return nil, errors.New(m.Errors)
}

func acquireTdnfRepoList(options *tdnf.Options, host string, token map[string]string) (*RepoListDesc, error) {
	resp, err := web.DispatchAndWait(http.MethodGet, host, "/api/v1/tdnf/repolist"+tdnfOptionsQuery(options), token, nil)
	if err != nil {
		fmt.Printf("Failed to acquire tdnf repolist: %v\n", err)
		return nil, err
	}

	m := RepoListDesc{}
	if err := json.Unmarshal(resp, &m); err != nil {
		os.Exit(1)
	}

	if m.Success {
		return &m, nil
	}

	return nil, errors.New(m.Errors)
}

func acquireTdnfInfoList(options *tdnf.Options, pkg string, host string, token map[string]string) (*InfoListDesc, error) {
	var path string
	if pkg != "" {
		path = "/api/v1/tdnf/info/" + pkg
	} else {
		path = "/api/v1/tdnf/info"
	}
	path = path + tdnfOptionsQuery(options)

	resp, err := web.DispatchAndWait(http.MethodGet, host, path, token, nil)
	if err != nil {
		return nil, err
	}

	m := InfoListDesc{}
	if err := json.Unmarshal(resp, &m); err != nil {
		fmt.Printf("Failed to decode json message: %v\n", err)
		os.Exit(1)
	}

	if m.Success {
		return &m, nil
	}

	return nil, errors.New(m.Errors)
}

func acquireTdnfSimpleCommand(options *tdnf.Options, cmd string, host string, token map[string]string) (*NilDesc, error) {
	var msg []byte

	msg, err := web.DispatchAndWait(http.MethodGet, host, "/api/v1/tdnf/"+cmd+tdnfOptionsQuery(options), token, nil)
	if err != nil {
		return nil, err
	}

	m := NilDesc{}
	if err := json.Unmarshal(msg, &m); err != nil {
		fmt.Printf("Failed to decode json message: %v\n", err)
		os.Exit(1)
	}

	if m.Success {
		return &m, nil
	}

	return nil, errors.New(m.Errors)
}

func tdnfClean(options *tdnf.Options, host string, token map[string]string) {
	_, err := acquireTdnfSimpleCommand(options, "clean", host, token)
	if err != nil {
		fmt.Printf("Failed execute tdnf clean: %v\n", err)
		return
	}
	fmt.Printf("package cache cleaned\n")
}

func tdnfList(options *tdnf.Options, pkg string, host string, token map[string]string) {
	l, err := acquireTdnfList(options, pkg, host, token)
	if err != nil {
		fmt.Printf("Failed to acquire tdnf list: %v\n", err)
		return
	}
	displayTdnfList(l)
}

func tdnfMakeCache(options *tdnf.Options, host string, token map[string]string) {
	_, err := acquireTdnfSimpleCommand(options, "makecache", host, token)
	if err != nil {
		fmt.Printf("Failed tdnf makecache: %v\n", err)
		return
	}
	fmt.Printf("package cache acquired\n")
}

func tdnfRepoList(options *tdnf.Options, host string, token map[string]string) {
	l, err := acquireTdnfRepoList(options, host, token)
	if err != nil {
		fmt.Printf("Failed to acquire tdnf repolist: %v\n", err)
		return
	}
	displayTdnfRepoList(l)
}

func tdnfInfoList(options *tdnf.Options, pkg string, host string, token map[string]string) {
	l, err := acquireTdnfInfoList(options, pkg, host, token)
	if err != nil {
		fmt.Printf("Failed to acquire tdnf info: %v\n", err)
		return
	}
	displayTdnfInfoList(l)
}
