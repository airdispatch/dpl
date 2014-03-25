package dpl

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
)

type PluginInstance struct {
	*Plugin
	Context interface{}
	Host    Host
}

type Plugin struct {
	Name          string            `xml:"name"`
	Path          string            `xml:"path"`
	Tag           []Tag             `xml:"tags>tag"`
	Tags          map[string]Tag    `xml:"-"`
	Action        []Action          `xml:"action"`
	Actions       map[string]Action `xml:"-"`
	DefaultAction string            `xml:"-"`
}

// Creates an Instance of a Plugin with a Host and Context
func (p *Plugin) CreateInstance(h Host, c interface{}) *PluginInstance {
	return &PluginInstance{p, c, h}
}

func (p *PluginInstance) RunActionWithContext(action string, message Message, user *User) (template.HTML, error) {
	var action_name string = action
	if action == "" {
		action_name = p.Plugin.DefaultAction
	}

	the_action := p.Plugin.Actions[action_name]

	t, err := template.New("plugin").Funcs(p.createFuncMap()).Parse(the_action.HTML)
	if err != nil {
		return "", err
	}

	var mc *MessageContext
	if message != nil {
		mc = &MessageContext{message, message.Sender()}
	}

	var buf bytes.Buffer
	err = t.ExecuteTemplate(&buf, "plugin",
		PluginContext{
			Host:    "DPL",
			Version: 0,
			Message: mc,
			User:    user,
		},
	)
	return template.HTML(buf.String()), err
}

// Method on PluginInstance called to render an action to a template
func (p *PluginInstance) RunAction(action string) (template.HTML, error) {
	return p.RunActionWithContext(action, nil, nil)
}

// Method on PluginInstance called when the MailServer receives a new message.
// It will return a list of tags that the message qualifies for.
func (p *PluginInstance) TagMessage(message Message) []Tag {
	output := make([]Tag, 0)
	for _, v := range p.Tags {
		tagged := true
		for _, field := range v.Fields {
			if message.Has(field.Name) {
				tagged = false
				break
			}
		}
		if tagged {
			output = append(output, v)
		}
	}
	return output
}

// Rendr a Plugin to a String (DEBUGGING USE)
func (p *Plugin) String() string {
	return fmt.Sprintf("%s (%s) %s \n %s", p.Name, p.Path, p.Tags, p.Action)
}

// Render a Tag to a String (DEBUGGING USE)
func (d Tag) String() string {
	output := "TAG: " + d.Name + " (" + d.Type + ")"
	for _, v := range d.Fields {
		output += "\n\t " + v.Name

		if !v.Optional {
			output += "\t(R)"
		}
	}
	return output
}

type Tag struct {
	Name      string  `xml:"name,attr"`
	Type      string  `xml:"type,attr"`
	OnReceive string  `xml:"onreceive,attr"`
	Fields    []Field `xml:"field"`
}

type Field struct {
	Name           string `xml:",innerxml"`
	Optional       bool   `xml:"optional,attr"`
	Representation string `xml:"repr,attr"` // Allows the plugin to access the value in other ways
}

type Action struct {
	HTML    string `xml:",innerxml"`
	Name    string `xml:"name,attr"`
	Default bool   `xml:"default,attr"`
}

func ParseDPLStream(r io.Reader) (*Plugin, error) {
	coder := xml.NewDecoder(r)

	var d Plugin
	err := coder.Decode(&d)
	if err != nil {
		return nil, err
	}

	return &d, verifyPlugin(&d)
}

func ParseDPL(r []byte) (*Plugin, error) {
	var d Plugin
	err := xml.Unmarshal(r, &d)
	if err != nil {
		return nil, err
	}

	return &d, verifyPlugin(&d)
}

func verifyPlugin(p *Plugin) error {
	if p.Path == "" {
		return errors.New("Plugin lacks a path.")
	}
	if p.Name == "" {
		p.Name = p.Path
	}

	if p.Tag == nil {
		return errors.New("Plugin lacks tags.")
	}
	for _, v := range p.Tag {
		if p.Tags == nil {
			p.Tags = make(map[string]Tag)
		}
		p.Tags[v.Name] = v
	}

	if p.Action == nil {
		return errors.New("Plugin lacks actions.")
	}
	for _, v := range p.Action {
		if p.Actions == nil {
			p.Actions = make(map[string]Action)
		}
		if v.Default {
			p.DefaultAction = v.Name
		}
		p.Actions[v.Name] = v
	}

	return nil
}
