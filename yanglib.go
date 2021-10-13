package yangtree

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

func (schema *SchemaNode) GetYangLibrary() DataNode {
	schema = schema.GetRootSchema()
	n, ok := schema.Annotation["ietf-yang-libary"]
	if ok {
		return n.(DataNode)
	}
	return nil
}

// <?xml version="1.0" encoding="UTF-8"?>
// <hello xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"
//   xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
//   <capabilities>
//     <capability>urn:ietf:params:netconf:base:1.0</capability>
//     <capability>urn:ietf:params:netconf:base:1.1</capability>
//     <capability>urn:ietf:params:netconf:capability:candidate:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:confirmed-commit:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:confirmed-commit:1.1</capability>
//     <capability>urn:ietf:params:netconf:capability:rollback-on-error:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:validate:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:validate:1.1</capability>
//     <capability>urn:ietf:params:netconf:capability:url:1.0?scheme=file</capability>
//     <capability>urn:ietf:params:netconf:capability:xpath:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:notification:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:interleave:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:partial-lock:1.0</capability>
//     <capability>urn:ietf:params:netconf:capability:with-defaults:1.0?basic-mode=report-all&amp;also-supported=trim,explicit,report-all-tagged</capability>
//     <capability>urn:ietf:params:xml:ns:yang:dot1x?module=dot1x&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr::ns:synce?module=esmc&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-alarms?module=hfr-alarms&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-ethernet?module=hfr-ethernet&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-events?module=hfr-events&amp;revision=2021-07-16</capability>
//     <capability>urn:mef:yang:hfr-flexport?module=hfr-flexport&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-gnmi?module=hfr-gnmi&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-hardware?module=hfr-hardware&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-ietf-spec?module=hfr-ietf-spec&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-if-aggregate?module=hfr-if-aggregate&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-lag?module=hfr-lag&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-link-oam?module=hfr-link-oam&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-lldp?module=hfr-lldp&amp;revision=2021-07-16</capability>
//     <capability>urn:mef:yang:hfr-mef?module=hfr-mef&amp;revision=2021-07-16</capability>
//     <capability>urn:mef:yang:hfr-mef-interfaces?module=hfr-mef-interfaces&amp;revision=2021-07-16</capability>
//     <capability>urn:mef:yang:hfr-mef-services?module=hfr-mef-services&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-mirror?module=hfr-mirror&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-performance-monitoring?module=hfr-performance-monitoring&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-ptp?module=hfr-ptp&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-qos?module=hfr-qos&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-radio-over-ethernet?module=hfr-radio-over-ethernet&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-system?module=hfr-system&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-system-logging?module=hfr-system-logging&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-tca?module=hfr-tca&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-time-sensitive-networking?module=hfr-time-sensitive-networking&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-trap?module=hfr-trap&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-twamp?module=hfr-twamp&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-vlans?module=hfr-vlans&amp;revision=2021-07-16</capability>
//     <capability>urn:hfr:ns:hfr-xwave?module=hfr-xwave&amp;revision=2021-07-16</capability>
//     <capability>urn:ietf:params:xml:ns:yang:iana-if-type?module=iana-if-type&amp;revision=2017-01-19</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-alarms?module=ietf-alarms&amp;revision=2017-10-30&amp;deviations=hfr-alarms</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-hardware?module=ietf-hardware&amp;revision=2018-03-13&amp;deviations=hfr-hardware</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-interfaces?module=ietf-interfaces&amp;revision=2018-02-20&amp;deviations=hfr-ietf-spec</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-ip?module=ietf-ip&amp;revision=2018-02-22&amp;deviations=hfr-ietf-spec</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-netconf-acm?module=ietf-netconf-acm&amp;revision=2018-02-14</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring?module=ietf-netconf-monitoring&amp;revision=2010-10-04</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-netconf-notifications?module=ietf-netconf-notifications&amp;revision=2012-02-06</capability>
//     <capability>urn:ietf:params:xml:ns:netconf:partial-lock:1.0?module=ietf-netconf-partial-lock&amp;revision=2009-10-19</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-netconf-with-defaults?module=ietf-netconf-with-defaults&amp;revision=2011-06-01</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-ptp?module=ietf-ptp&amp;revision=2019-05-07&amp;deviations=hfr-ptp</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-system?module=ietf-system&amp;revision=2014-08-06&amp;features=radius,authentication,local-users,radius-authentication,ntp,timezone-name&amp;deviations=hfr-system</capability>
//     <capability>urn:ietf:params:xml:ns:yang:ietf-yang-library?module=ietf-yang-library&amp;revision=2016-06-21</capability>
//     <capability>urn:ietf:params:xml:ns:netmod:notification?module=nc-notifications&amp;revision=2020-01-10</capability>
//     <capability>urn:ietf:params:xml:ns:netconf:notification:1.0?module=notifications&amp;revision=2008-07-14</capability>
//     <capability>http://netconfcentral.org/ns/yuma-mysession?module=yuma-mysession&amp;revision=2010-05-10</capability>
//     <capability>http://netconfcentral.org/ns/yuma-ncx?module=yuma-ncx&amp;revision=2018-01-30</capability>
//     <capability>http://netconfcentral.org/ns/yuma-proc?module=yuma-proc&amp;revision=2012-10-10</capability>
//     <capability>http://netconfcentral.org/ns/yuma-time-filter?module=yuma-time-filter&amp;revision=2012-11-15</capability>
//     <capability>http://yuma123.org/ns/yuma123-mysession-cache?module=yuma123-mysession-cache&amp;revision=2018-11-12</capability>
//     <capability>urn:ietf:params:xml:ns:netconf:base:1.0?module=ietf-netconf&amp;revision=2011-06-01&amp;features=candidate,confirmed-commit,rollback-on-error,validate,url,xpath</capability>
//     <capability>http://yuma123.org/ns/yuma123-system?module=yuma123-system&amp;revision=2020-02-03</capability>
//     <capability>urn:ietf:params:netconf:capability:yang-library:1.0?revision=2016-06-21&amp;module-set-id=aea9fde437609ddbe8f53a999dff3395c77f446e</capability>
//   </capabilities>
//   <session-id>1</session-id>
// </hello>

func Capabilities(namespace, modulename, revision, feature string) {
	// capabilities format
	// {namespace}?module={modulename}&revision={revision}
	// {namespace}?module={modulename}&revision={revision}&features={feature},{feature},..
	// {namespace}?module={modulename}&revision={revision}&deviations={deviation},{deviation},..
}

func checkAccessableObjects(p yang.Node, nodelist interface{}) bool {
	v := reflect.ValueOf(nodelist)
	for i := 0; i < v.Len(); i++ {
		vv := v.Index(i)
		node := vv.Interface()
		if p == node.(yang.Node).ParentNode() {
			return true
		}
	}
	return false
}

func getConformanceType(m *yang.Module, excluded []string) (conformancetype string) {
	for i := range excluded {
		if excluded[i] == m.Name {
			return "import"
		}
	}
	// check the module has protocol-accessible objects.
	implement := false
	if len(m.Augment) > 0 {
		implement = true
	}
	if len(m.Deviation) > 0 {
		implement = true
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Anydata)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Anyxml)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Container)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Choice)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.List)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Uses)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Leaf)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.LeafList)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.RPC)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Notification)
	}
	if !implement {
		implement = checkAccessableObjects(m, m.Anydata)
	}

	if implement {
		conformancetype = "implement"
	} else {
		conformancetype = "import"
	}
	return
}

// Module set ID
var moduleSetNum int

func loadYanglibrary(rootschema *SchemaNode, excluded []string) error {
	modulemap := rootschema.Modules.Modules
	moduleSetNum++
	ylib := modulemap["ietf-yang-library"]
	if ylib == nil {
		if rootschema.Option.YANGLibrary2016 ||
			rootschema.Option.YANGLibrary2019 {
			return fmt.Errorf("yanglib: ietf-yang-library is not loaded")
		}
		return nil
	}
	var err error
	var top DataNode
	switch ylib.Current() {
	case "2019-01-04":
		moduleSetName := fmt.Sprintf("set-%d", moduleSetNum)
		// load the previous module set
		if rootschema.Option != nil && rootschema.Option.SchemaSetName != "" {
			moduleSetName = rootschema.Option.SchemaSetName
		}
		top, err = NewDataNode(rootschema.GetSchema("yang-library"))
		if err != nil {
			return fmt.Errorf(`yanglib: %q not found`, "yang-library")
		}
		var mods []*yang.Module
		for _, m := range modulemap {
			mods = append(mods, m)
		}
		sort.Slice(mods, func(i, j int) bool {
			if mods[i].Name < mods[j].Name {
				return true
			} else if mods[i].Name == mods[j].Name {
				return mods[i].Current() < mods[j].Current()
			}
			return true
		})
		for _, m := range modulemap {
			if m.BelongsTo != nil {
				continue
			}
			name, revision, namespace := m.Name, m.Current(), ""
			if m.Namespace != nil {
				namespace = m.Namespace.Name
			}

			// module
			listname := "module"
			isImport := getConformanceType(m, excluded)
			if isImport == "import" {
				listname = "import-only-module"
				err := Set(top, fmt.Sprintf(
					"module-set[name=%s]/%s[name=%s][revision=%s]",
					moduleSetName, listname, name, revision),
					fmt.Sprintf(`{"namespace":%q}`, namespace))
				if err != nil {
					return fmt.Errorf("yanglib: unable to add module %q: %v", name, err)
				}
			} else {
				err := Set(top, fmt.Sprintf(
					"module-set[name=%s]/%s[name=%s][revision=%s]",
					moduleSetName, listname, name, revision),
					fmt.Sprintf(`{"namespace":%q}`, namespace))
				if err != nil {
					return fmt.Errorf("yanglib: unable to add module %q: %v", name, err)
				}
				// feature
				for i := range m.Feature {
					p := fmt.Sprintf(
						"module-set[name=%s]/%s[name=%s][revision=%s]/feature[.=%s]",
						moduleSetName, listname, name, revision, m.Feature[i].Name)
					err = Set(top, p, m.Feature[i].Name)
					if err != nil {
						return fmt.Errorf("yanglib: unable to add module %q: %v", name, err)
					}
				}
				// deviation
				for i := range m.Deviation {
					// fmt.Println(m.Name, m.Deviation[i].Name)
					pathnode, err := ParsePath(&m.Deviation[i].Name)
					if err != nil || len(pathnode) == 0 {
						return fmt.Errorf("yanglib: can not find target node %q to deviate", m.Deviation[i].Name)
					}
					prefix := pathnode[len(pathnode)-1].Prefix
					target := yang.FindModuleByPrefix(m, prefix)
					if target == nil {
						target = modulemap[prefix]
						if target == nil {
							return fmt.Errorf("yanglib: deviation schema %q not found", m.Deviation[i].Name)
						}
					}
					p := fmt.Sprintf("module-set[name=%s]/%s[name=%s][revision=%s]/deviation[.=%s]",
						moduleSetName, listname, target.Name, target.Current(), name)
					if n, err := Find(top, p); err == nil && len(n) == 0 {
						err = Set(top, p, name)
						if err != nil {
							return fmt.Errorf("yanglib: unable to add deviation module to %q: %v", name, err)
						}
					}
				}
			}

			// submodule
			for i := range m.Include {
				sm := m.Include[i].Module
				if sm != nil {
					subname, subrevision := sm.Name, sm.Current()
					err := Set(top, fmt.Sprintf(
						"module-set[name=%s]/%s[name=%s]/submodule[name=%s][revision=%s]",
						moduleSetName, listname, name, subname, subrevision), "")
					if err != nil {
						return fmt.Errorf("yanglib: unable to add submodule %q: %v", name, err)
					}
				}
			}
		}
		var contentId strings.Builder
		b, _ := MarshalYAML(top, InternalFormat{})
		// fmt.Println(string(b))
		h := sha1.New()
		io.WriteString(h, string(b))
		b = h.Sum(nil)
		encoder := base64.NewEncoder(base64.StdEncoding, &contentId)
		encoder.Write(b)
		encoder.Close()
		// fmt.Println(contentId.String())
		if err := Set(top, "content-id", contentId.String()); err != nil {
			return fmt.Errorf("yanglib: content-id generation error: %v", err)
		}
	case "2016-06-21":
		top, err = NewDataNode(rootschema.GetSchema("modules-state"))
		if err != nil {
			return fmt.Errorf(`yanglib: %q not found`, "modules-state")
		}
		for _, m := range modulemap {
			name, revision, namespace := m.Name, m.Current(), ""
			if m.Namespace != nil {
				namespace = m.Namespace.Name
			}
			// module
			if m.BelongsTo == nil {
				err := Set(top, fmt.Sprintf("module[name=%s][revision=%s]", name, revision),
					fmt.Sprintf(`{"namespace":%q,"conformance-type":%q}`, namespace, getConformanceType(m, excluded)))
				if err != nil {
					return fmt.Errorf("yanglib: unable to add module %q: %v", name, err)
				}
			}
			// feature
			for i := range m.Feature {
				p := fmt.Sprintf("module[name=%s][revision=%s]/feature[.=%s]", name, revision, m.Feature[i].Name)
				if n, err := Find(top, p); err == nil && len(n) == 0 {
					err = Set(top, p, m.Feature[i].Name)
					if err != nil {
						return fmt.Errorf("yanglib: unable to add deviation module to %q: %v", name, err)
					}
				}
			}
			// deviation
			for i := range m.Deviation {
				pathnode, err := ParsePath(&m.Deviation[i].Name)
				if err != nil || len(pathnode) == 0 {
					return fmt.Errorf("yanglib: can not find target node %q to deviate", m.Deviation[i].Name)
				}
				prefix := pathnode[len(pathnode)-1].Prefix
				target := yang.FindModuleByPrefix(m, prefix)
				if target == nil {
					target = modulemap[prefix]
					if target == nil {
						return fmt.Errorf("yanglib: deviation schema %q not found", m.Deviation[i].Name)
					}
				}
				err = Set(top, fmt.Sprintf("module[name=%s][revision=%s]/deviation[name=%s][revision=%s]",
					target.Name, target.Current(), name, revision), "")
				if err != nil {
					return fmt.Errorf("yanglib: unable to add deviation module to %q: %v", name, err)
				}
			}
			// submodule
			for i := range m.Include {
				sm := m.Include[i].Module
				if sm != nil {
					subname, subrevision := sm.Name, sm.Current()
					err := Set(top, fmt.Sprintf("module[name=%s][revision=%s]/submodule[name=%s][revision=%s]",
						name, revision, subname, subrevision), "")
					if err != nil {
						return fmt.Errorf("yanglib: unable to add submodule %q: %v", name, err)
					}
				}
			}
		}
		var moduleSetId strings.Builder
		b, _ := MarshalYAML(top, InternalFormat{})
		// fmt.Println(string(b))
		h := sha1.New()
		io.WriteString(h, string(b))
		b = h.Sum(nil)
		encoder := base64.NewEncoder(base64.StdEncoding, &moduleSetId)
		encoder.Write(b)
		encoder.Close()
		// fmt.Println(moduleSetId.String())
		if err := Set(top, "module-set-id", moduleSetId.String()); err != nil {
			return fmt.Errorf("yanglib: module-set-id generation error: %v", err)
		}
	}
	if top != nil {
		if rootschema.Annotation == nil {
			rootschema.Annotation = make(map[string]interface{})
		}
		rootschema.Annotation["ietf-yang-libary"] = top
	}
	return nil
}
