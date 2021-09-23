package yangtree

import "fmt"

// NETCONF error (https://datatracker.ietf.org/doc/html/rfc6241#appendix-A)

type ErrorTag int

const (
	ETagInUse ErrorTag = iota
	ETagInvalidValue
	ETagTooBig
	ETagMissingAttribute
	ETagBadAttribute
	ETagUnknownAttribute
	ETagMissingElement
	ETagBadElement
	ETagUnknownElement
	ETagUnknownNamespace
	ETagAccessDenied
	ETagLockDenied
	ETagResourceDenied
	ETagRollbackFailed
	ETagDataExists
	ETagDataMissing
	ETagOperationNotSupported
	ETagOperationFailed
	ETagPartialOperation
	ETagMarlformedMessage

	EAppTagUnknownError ErrorTag = iota + 100
	EAppTagDataNodeMissing
	EAppTagDataNodeExists
	EAppTagInvalidArg
)

func (et ErrorTag) String() string {
	switch et {
	case ETagInUse:
		return "in-use"
	case ETagInvalidValue:
		return "invalid-value"
	case ETagTooBig:
		return "too-big"
	case ETagMissingAttribute:
		return "missing-attribute"
	case ETagBadAttribute:
		return "bad-attribute"
	case ETagUnknownAttribute:
		return "unknown-attribute"
	case ETagMissingElement:
		return "missing-element"
	case ETagBadElement:
		return "bad-element"
	case ETagUnknownElement:
		return "unknown-element"
	case ETagUnknownNamespace:
		return "unknown-namespace"
	case ETagAccessDenied:
		return "access-denied"
	case ETagLockDenied:
		return "lock-denied"
	case ETagResourceDenied:
		return "resource-denied"
	case ETagRollbackFailed:
		return "rollback-failed"
	case ETagDataExists:
		return "data-exists"
	case ETagDataMissing:
		return "data-missing"
	case ETagOperationNotSupported:
		return "operation-not-supported"
	case ETagOperationFailed:
		return "operation-failed"
	case ETagPartialOperation:
		return "partial-operation"
	case ETagMarlformedMessage:
		return "marlformed-message"
	case EAppTagUnknownError:
		return "unknown-error"
	case EAppTagDataNodeMissing:
		return "data-node-missing"
	case EAppTagDataNodeExists:
		return "data-node-exists"
	case EAppTagInvalidArg:
		return "invalid-argument"
	default:
		return "unknown"
	}
}

type ErrorType int

const (
	ETypeApplication ErrorType = iota
	ETypeProtocol
	ETypeRPC
	ETypeTransport
)

func (et ErrorType) String() string {
	switch et {
	case ETypeApplication:
		return "application"
	case ETypeProtocol:
		return "protocol"
	case ETypeRPC:
		return "rpc"
	case ETypeTransport:
		return "transport"
	default:
		return "unknown"
	}
}

type YError struct {
	ErrorTag
	// ErrorAppTag ErrorTag
	ErrorType
	ErrorMessage string
}

func (ye *YError) Error() string {
	if ye == nil {
		return ""
	}
	return "[" + ye.ErrorTag.String() + "] " + ye.ErrorMessage
}

func NewError(etag ErrorTag, eMessage string) *YError {
	return &YError{
		ErrorTag:     etag,
		ErrorType:    ETypeApplication,
		ErrorMessage: eMessage,
	}
}

func NewErrorf(etag ErrorTag, eMessage string, arg ...interface{}) *YError {
	return &YError{
		ErrorTag:     etag,
		ErrorType:    ETypeApplication,
		ErrorMessage: fmt.Sprintf(eMessage, arg...),
	}
}

func Errorf(etag ErrorTag, eMessage string, arg ...interface{}) *YError {
	return &YError{
		ErrorTag:     etag,
		ErrorType:    ETypeApplication,
		ErrorMessage: fmt.Sprintf(eMessage, arg...),
	}
}

// 4.3.  <rpc-error> Element

//    The <rpc-error> element is sent in <rpc-reply> messages if an error
//    occurs during the processing of an <rpc> request.

//    If a server encounters multiple errors during the processing of an
//    <rpc> request, the <rpc-reply> MAY contain multiple <rpc-error>
//    elements.  However, a server is not required to detect or report more
//    than one <rpc-error> element, if a request contains multiple errors.
//    A server is not required to check for particular error conditions in
//    a specific sequence.  A server MUST return an <rpc-error> element if
//    any error conditions occur during processing.

//    A server MUST NOT return application-level- or data-model-specific
//    error information in an <rpc-error> element for which the client does
//    not have sufficient access rights.

//    The <rpc-error> element includes the following information:

//    error-type:  Defines the conceptual layer that the error occurred.
//       Enumeration.  One of:

//       *  transport (layer: Secure Transport)

//       *  rpc (layer: Messages)

//       *  protocol (layer: Operations)

//       *  application (layer: Content)

//    error-tag:  Contains a string identifying the error condition.  See
//       Appendix A for allowed values.

//    error-severity:  Contains a string identifying the error severity, as
//       determined by the device.  One of:

//       *  error

//       *  warning

//       Note that there are no <error-tag> values defined in this document
//       that utilize the "warning" enumeration.  This is reserved for
//       future use.

//    error-app-tag:  Contains a string identifying the data-model-specific
//       or implementation-specific error condition, if one exists.  This
//       element will not be present if no appropriate application error-
//       tag can be associated with a particular error condition.  If a
//       data-model-specific and an implementation-specific error-app-tag
//       both exist, then the data-model-specific value MUST be used by the
//       server.

//    error-path:  Contains the absolute XPath [W3C.REC-xpath-19991116]
//       expression identifying the element path to the node that is
//       associated with the error being reported in a particular
//       <rpc-error> element.  This element will not be present if no
//       appropriate payload element or datastore node can be associated
//       with a particular error condition.

//       The XPath expression is interpreted in the following context:

//       *  The set of namespace declarations are those in scope on the
//          <rpc-error> element.

//       *  The set of variable bindings is empty.

//       *  The function library is the core function library.

//       The context node depends on the node associated with the error
//       being reported:

//       *  If a payload element can be associated with the error, the
//          context node is the rpc request's document node (i.e., the
//          <rpc> element).

//       *  Otherwise, the context node is the root of all data models,
//          i.e., the node that has the top-level nodes from all data
//          models as children.

//    error-message:  Contains a string suitable for human display that
//       describes the error condition.  This element will not be present
//       if no appropriate message is provided for a particular error
//       condition.  This element SHOULD include an "xml:lang" attribute as
//       defined in [W3C.REC-xml-20001006] and discussed in [RFC3470].

//    error-info:  Contains protocol- or data-model-specific error content.
//       This element will not be present if no such error content is
//       provided for a particular error condition.  The list in Appendix A
//       defines any mandatory error-info content for each error.  After
//       any protocol-mandated content, a data model definition MAY mandate
//       that certain application-layer error information be included in
//       the error-info container.  An implementation MAY include
//       additional elements to provide extended and/or implementation-
//       specific debugging information.

//    Appendix A enumerates the standard NETCONF errors.

//    Example:  An error is returned if an <rpc> element is received
//       without a "message-id" attribute.  Note that only in this case is
//       it acceptable for the NETCONF peer to omit the "message-id"
//       attribute in the <rpc-reply> element.

//      <rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
//        <get-config>
//          <source>
//            <running/>
//          </source>
//        </get-config>
//      </rpc>

//      <rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
//        <rpc-error>
//          <error-type>rpc</error-type>
//          <error-tag>missing-attribute</error-tag>
//          <error-severity>error</error-severity>
//          <error-info>
//            <bad-attribute>message-id</bad-attribute>
//            <bad-element>rpc</bad-element>
//          </error-info>
//        </rpc-error>
//      </rpc-reply>

//    The following <rpc-reply> illustrates the case of returning multiple
//    <rpc-error> elements.

//    Note that the data models used in the examples in this section use
//    the <name> element to distinguish between multiple instances of the
//    <interface> element.

//      <rpc-reply message-id="101"
//        xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"
//        xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
//        <rpc-error>
//          <error-type>application</error-type>
//          <error-tag>invalid-value</error-tag>
//          <error-severity>error</error-severity>
//          <error-path xmlns:t="http://example.com/schema/1.2/config">
//            /t:top/t:interface[t:name="Ethernet0/0"]/t:mtu
//          </error-path>
//          <error-message xml:lang="en">
//            MTU value 25000 is not within range 256..9192
//          </error-message>
//        </rpc-error>
//        <rpc-error>
//          <error-type>application</error-type>
//          <error-tag>invalid-value</error-tag>
//          <error-severity>error</error-severity>
//          <error-path xmlns:t="http://example.com/schema/1.2/config">
//            /t:top/t:interface[t:name="Ethernet1/0"]/t:address/t:name
//          </error-path>
//          <error-message xml:lang="en">
//            Invalid IP address for interface Ethernet1/0
//          </error-message>
//        </rpc-error>
//      </rpc-reply>
