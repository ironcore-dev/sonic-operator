// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"io"
	"strings"

	errors "github.com/ironcore-dev/sonic-operator/internal/agent/errors"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"

	"github.com/jedib0t/go-pretty/v6/table"
)

type Renderer interface {
	Render(info string, v any) error
}

// Basic renderer
type BasicRenderer struct {
	w io.Writer
}

func NewBasicRenderer(w io.Writer) *BasicRenderer {
	return &BasicRenderer{w: w}
}

func (r *BasicRenderer) Render(info string, v any) error {

	var objs []agent.Object
	switch v := v.(type) {
	case agent.Object:
		objs = []agent.Object{v}
	case agent.List:
		objs = v.GetItems()
	default:
		return fmt.Errorf("unsupported type %T for rendering", v)
	}

	if info != "" {
		if _, err := fmt.Fprintln(r.w, info); err != nil {
			return err
		}
	}

	for _, obj := range objs {
		if err := r.renderObject(obj); err != nil {
			return err
		}
	}
	return nil
}

func (r *BasicRenderer) renderObject(obj agent.Object) error {
	var parts []string
	if kind := obj.GetKind(); kind != "" {
		parts = append(parts, fmt.Sprintf("%s/%s", strings.ToLower(kind), obj.GetName()))
	} else {
		parts = append(parts, obj.GetName())
	}

	_, err := fmt.Fprintf(r.w, "%s\n", strings.Join(parts, " "))
	return err
}

// Table renderer
type TableData struct {
	Headers []any
	Rows    [][]any
}

type TableRenderer struct {
	w              io.Writer
	tableConverter TableConverter
}

func NewTableRenderer(w io.Writer, converter TableConverter) *TableRenderer {
	return &TableRenderer{w, converter}
}

func (t *TableRenderer) Render(info string, v any) error {
	data, err := t.tableConverter.ConvertToTable(v)
	if err != nil {
		return err
	}

	// Print info first
	if info != "" {
		if _, err := fmt.Fprintln(t.w, info); err != nil {
			return err
		}
	}

	tw := table.NewWriter()
	tw.SetStyle(tableStyle)
	tw.SetOutputMirror(t.w)

	tw.AppendHeader(data.Headers)
	for _, row := range data.Rows {
		tw.AppendRow(row)
	}

	tw.Render()
	return nil
}

type TableConverter interface {
	ConvertToTable(v any) (*TableData, error)
}

type defaultTableConverter struct{}

var DefaultTableConverter = defaultTableConverter{}

func (t defaultTableConverter) ConvertToTable(v any) (*TableData, error) {
	switch obj := v.(type) {
	case *agent.SwitchDevice:
		return t.deviceToTable(*obj)
	case *agent.Interface:
		return t.interfaceToTable([]agent.Interface{*obj})
	case *agent.InterfaceList:
		return t.interfaceToTable(obj.Items)
	case *agent.PortList:
		return t.portToTable(obj.Items)
	case *agent.Port:
		return t.portToTable([]agent.Port{*obj})
	case *agent.InterfaceNeighbor:
		return t.interfaceNeighborToTable([]agent.InterfaceNeighbor{*obj})
	}
	return nil, fmt.Errorf("unsupported type %T for table conversion", v)
}

func (t defaultTableConverter) deviceToTable(device agent.SwitchDevice) (*TableData, error) {
	headers := []any{"Name", "MAC Address", "HW SKU", "Sonic OS Version", "ASIC Type", "Readiness"}
	rows := make([][]any, 1)
	rows[0] = []any{
		device.GetName(),
		device.LocalMacAddress,
		device.Hwsku,
		device.SonicOSVersion,
		device.AsicType,
		device.Readiness,
	}

	return &TableData{Headers: headers, Rows: rows}, nil
}

func (t defaultTableConverter) interfaceToTable(ifaces []agent.Interface) (*TableData, error) {
	headers := []any{"Name", "MAC Address", "Operation Status", "Admin Status"}
	rows := make([][]any, len(ifaces))

	for _, iface := range ifaces {
		rows = append(rows, []any{
			iface.Name,
			iface.MacAddress,
			iface.OperationStatus,
			iface.AdminStatus,
		})
	}

	return &TableData{Headers: headers, Rows: rows}, nil
}

func (t defaultTableConverter) portToTable(ports []agent.Port) (*TableData, error) {
	headers := []any{"Name", "Alias"}
	rows := make([][]any, len(ports))

	for _, port := range ports {
		rows = append(rows, []any{
			port.Name,
			port.Alias,
		})
	}

	return &TableData{Headers: headers, Rows: rows}, nil
}

func (t defaultTableConverter) interfaceNeighborToTable(neighbors []agent.InterfaceNeighbor) (*TableData, error) {

	headers := []any{"Neighbor Name", "Handle", "MAC Address"}

	rows := make([][]any, 1)
	rows[0] = []any{
		neighbors[0].SystemName,
		neighbors[0].Handle,
		neighbors[0].MacAddress,
	}

	return &TableData{Headers: headers, Rows: rows}, nil
}

var (
	lightBoxStyle = table.BoxStyle{
		BottomLeft:       "",
		BottomRight:      "",
		BottomSeparator:  "",
		EmptySeparator:   " ",
		Left:             "",
		LeftSeparator:    "",
		MiddleHorizontal: "",
		MiddleSeparator:  "",
		MiddleVertical:   " ",
		PaddingLeft:      " ",
		PaddingRight:     " ",
		PageSeparator:    "\n",
		Right:            "",
		RightSeparator:   "",
		TopLeft:          "",
		TopRight:         "",
		TopSeparator:     "",
		UnfinishedRow:    "",
	}
	tableStyle = table.Style{Box: lightBoxStyle}
)

type RenderFunc func(w io.Writer) Renderer

type RendererFactory struct {
	renderFuncMap map[string]RenderFunc
}

func NewRendererFactory() *RendererFactory {
	return &RendererFactory{
		renderFuncMap: make(map[string]RenderFunc),
	}
}

func (f *RendererFactory) registerDefaultRenderers() error {
	if err := f.RegisterRenderer("basic", func(w io.Writer) Renderer {
		return NewBasicRenderer(w)
	}); err != nil {
		return err
	}

	if err := f.RegisterRenderer("table", func(w io.Writer) Renderer {
		return NewTableRenderer(w, DefaultTableConverter)
	}); err != nil {
		return err
	}

	return nil
}

func (f *RendererFactory) RegisterRenderer(name string, renderFunc RenderFunc) error {
	if _, exists := f.renderFuncMap[name]; exists {
		return fmt.Errorf("renderer %s already registered", name)
	}
	f.renderFuncMap[name] = renderFunc
	return nil
}

func (f *RendererFactory) GetRenderer(name string, w io.Writer) (Renderer, error) {
	renderFunc, exists := f.renderFuncMap[name]
	if !exists {
		return nil, fmt.Errorf("renderer %s not found", name)
	}
	return renderFunc(w), nil
}

type PrintRenderer interface {
	Print(info string, w io.Writer, v any) error
}

type DefaultPrintRender struct {
	RendererType    string
	RendererFactory *RendererFactory
}

func NewDefaultPrintRender(rendererType string) *DefaultPrintRender {
	f := NewRendererFactory()
	if err := f.registerDefaultRenderers(); err != nil {
		panic(fmt.Sprintf("failed to register default renderers: %v", err))
	}
	return &DefaultPrintRender{RendererType: rendererType, RendererFactory: f}
}

func (p DefaultPrintRender) RenderObject(info string, w io.Writer, obj agent.Object) *agent.Status {
	if obj.GetStatus().Code != 0 {
		info = fmt.Sprintf("server error: %d, %s", obj.GetStatus().Code, obj.GetStatus().Message)
		if p.RendererType == "table" {
			p.RendererType = "basic"
		}
	}
	renderer, err := p.RendererFactory.GetRenderer(p.RendererType, w)
	if err != nil {
		return agent.NewErrorStatus(errors.BAD_REQUEST, fmt.Sprintf("error creating renderer: %v", err))
	}
	if err := renderer.Render(info, obj); err != nil {
		return agent.NewErrorStatus(errors.CLIENT_ERROR, fmt.Sprintf("error rendering %s: %v", obj.GetKind(), err))
	}
	if obj.GetStatus().Code != 0 {
		return agent.NewErrorStatus(obj.GetStatus().Code, obj.GetStatus().Message)
	}
	return nil
}

func (p DefaultPrintRender) RenderList(info string, w io.Writer, list agent.List) *agent.Status {
	if list.GetStatus().Code != 0 {
		info = fmt.Sprintf("server error: %d, %s", list.GetStatus().Code, list.GetStatus().Message)
		if p.RendererType == "table" {
			p.RendererType = "basic"
		}
	}
	renderer, err := p.RendererFactory.GetRenderer(p.RendererType, w)
	if err != nil {
		return agent.NewErrorStatus(errors.CLIENT_ERROR, fmt.Sprintf("error creating renderer: %v", err))
	}
	if err := renderer.Render(info, list); err != nil {
		return agent.NewErrorStatus(errors.CLIENT_ERROR, fmt.Sprintf("error rendering list: %v", err))
	}

	if list.GetStatus().Code != 0 {
		return agent.NewErrorStatus(list.GetStatus().Code, list.GetStatus().Message)
	}

	return nil
}

func (d *DefaultPrintRender) Print(info string, w io.Writer, v any) error {
	switch v := v.(type) {
	case agent.Object:
		status := d.RenderObject(info, w, v)
		if status != nil && status.Code != 0 {
			return fmt.Errorf("error rendering object: %s", status.Message)
		}
	case agent.List:
		status := d.RenderList(info, w, v)
		if status != nil && status.Code != 0 {
			return fmt.Errorf("error rendering list: %s", status.Message)
		}
	default:
		return fmt.Errorf("unsupported type %T for printing", v)
	}

	return nil
}
