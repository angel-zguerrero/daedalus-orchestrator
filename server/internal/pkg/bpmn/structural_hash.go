package bpmn

import "crypto/sha256"
import "encoding/hex"
import "encoding/xml"
import "bytes"
import "sort"
import "strings"

// ComputeStructuralHash computes a SHA-256 hash of the BPMN XML content ignoring
// visual/diagram elements (such as bpmndi:BPMNDiagram, dc:Bounds, di:waypoint, etc.).
func ComputeStructuralHash(xmlData []byte) (string, error) {
	if len(bytes.TrimSpace(xmlData)) == 0 {
		return "", nil
	}

	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	var canonicalBuf bytes.Buffer

	skipDepth := 0

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			nameLocal := t.Name.Local
			nameSpace := t.Name.Space

			isDiagramElement := isBPMNDiagramElement(nameSpace, nameLocal)

			if skipDepth > 0 || isDiagramElement {
				skipDepth++
				continue
			}

			canonicalBuf.WriteString("<" + nameLocal)

			// Sort attributes for deterministic output
			attrs := filterAndSortAttributes(t.Attr)
			for _, attr := range attrs {
				canonicalBuf.WriteString(" " + attr.Name.Local + "=\"" + strings.TrimSpace(attr.Value) + "\"")
			}
			canonicalBuf.WriteString(">")

		case xml.EndElement:
			if skipDepth > 0 {
				skipDepth--
				continue
			}
			canonicalBuf.WriteString("</" + t.Name.Local + ">")

		case xml.CharData:
			if skipDepth > 0 {
				continue
			}
			trimmed := strings.TrimSpace(string(t))
			if len(trimmed) > 0 {
				canonicalBuf.WriteString(trimmed)
			}
		}
	}

	hash := sha256.Sum256(canonicalBuf.Bytes())
	return hex.EncodeToString(hash[:]), nil
}

func isBPMNDiagramElement(space, local string) bool {
	spaceLower := strings.ToLower(space)
	localLower := strings.ToLower(local)

	if strings.Contains(spaceLower, "bpmndi") || strings.Contains(spaceLower, "/dc") || strings.Contains(spaceLower, "/di") {
		return true
	}
	if strings.HasPrefix(localLower, "bpmnd") || strings.HasPrefix(localLower, "bpmnp") || strings.HasPrefix(localLower, "bpmns") || strings.HasPrefix(localLower, "bpmne") {
		return true
	}
	switch localLower {
	case "bpmndiagram", "bpmnplane", "bpmnshape", "bpmnedge", "bounds", "waypoint", "label":
		return true
	}
	return false
}

func filterAndSortAttributes(attrs []xml.Attr) []xml.Attr {
	var filtered []xml.Attr
	for _, a := range attrs {
		spaceLower := strings.ToLower(a.Name.Space)
		localLower := strings.ToLower(a.Name.Local)
		if strings.Contains(spaceLower, "bpmndi") || strings.Contains(spaceLower, "/dc") || strings.Contains(spaceLower, "/di") {
			continue
		}
		if localLower == "x" || localLower == "y" || localLower == "width" || localLower == "height" {
			continue
		}
		filtered = append(filtered, a)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name.Local < filtered[j].Name.Local
	})
	return filtered
}
