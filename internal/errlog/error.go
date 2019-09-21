package errlog

import (
	"fmt"
)

// ErrorLog ...
type ErrorLog struct {
	Errors []*Error
}

// ErrorCode ...
type ErrorCode int

const (
	// ErrorUnreachable ...
	ErrorUnreachable ErrorCode = 1 + iota
	// ErrorUninitializedVariable ...
	ErrorUninitializedVariable
	// ErrorNamedGroupMerge ...
	ErrorNamedGroupMerge
	// ErrorScopedGroupMerge ...
	ErrorScopedGroupMerge
	// ErrorIllegalRune ...
	ErrorIllegalRune
	// ErrorIllegalString ...
	ErrorIllegalString
	// ErrorIllegalNumber ...
	ErrorIllegalNumber
	// ErrorIllegalCharacter ...
	ErrorIllegalCharacter
	// ErrorExpectedToken ...
	ErrorExpectedToken
	// ErrorUnexpectedEOF ...
	ErrorUnexpectedEOF
	// ErrorUnknownType ...
	ErrorUnknownType
	// ErrorUnknownNamespace ...
	ErrorUnknownNamespace
	// ErrorArraySizeInteger ...
	ErrorArraySizeInteger
	// ErrorCyclicTypeDefinition ...
	ErrorCyclicTypeDefinition
	// ErrorStructBaseType ...
	ErrorStructBaseType
	// ErrorStructSingleBaseType ...
	ErrorStructSingleBaseType
	// ErrorStructDuplicateField ...
	ErrorStructDuplicateField
	// ErrorDuplicateTypeName ...
	ErrorDuplicateTypeName
	// ErrorStructDuplicateInterface ...
	ErrorStructDuplicateInterface
	// ErrorInterfaceDuplicateInterface ...
	ErrorInterfaceDuplicateInterface
	// ErrorInterfaceBaseType ...
	ErrorInterfaceBaseType
	// ErrorInterfaceDuplicateFunc ...
	ErrorInterfaceDuplicateFunc
	// ErrorDuplicateParameter ...
	ErrorDuplicateParameter
	// ErrorUnnamedParameter ...
	ErrorUnnamedParameter
	// ErrorDuplicateScopeName ...
	ErrorDuplicateScopeName
	// ErrorNotANamespace ...
	ErrorNotANamespace
	// ErrorUnknownIdentifier ...
	ErrorUnknownIdentifier
	// ErrorTypeCannotHaveFunc ...
	ErrorTypeCannotHaveFunc
	// ErrorNotAGenericType ...
	ErrorNotAGenericType
	// ErrorWrongTypeArgumentCount ...
	ErrorWrongTypeArgumentCount
	// ErrorMalformedPackagePath ...
	ErrorMalformedPackagePath
	// ErrorPackageNotFound ...
	ErrorPackageNotFound
	// ErrorNameNotExported ...
	ErrorNameNotExported
	// ErrorTypeCannotBeInstantiated ...
	ErrorTypeCannotBeInstantiated
	// ErrorNumberOutOfRange ...
	ErrorNumberOutOfRange
	// ErrorComponentTwice ...
	ErrorComponentTwice
	// ErrorIncompatibleTypes ...
	ErrorIncompatibleTypes
	// ErrorIncompatibleTypeForOp ...
	ErrorIncompatibleTypeForOp
	// ErrorGenericMustBeInstantiated ...
	ErrorGenericMustBeInstantiated
	// ErrorNoValueType ...
	ErrorNoValueType
	// ErrorTemporaryNotAssignable ...
	ErrorTemporaryNotAssignable
	// ErrorNotMutable ...
	ErrorNotMutable
	// ErrorVarWithoutType ...
	ErrorVarWithoutType
	// ErrorExpectedVariable ...
	ErrorExpectedVariable
	// ErrorWrongMutGroupOrder ...
	ErrorWrongMutGroupOrder
	// AssignmentValueCountMismatch ...
	AssignmentValueCountMismatch
	// ErrorNoNewVarsInAssignment ...
	ErrorNoNewVarsInAssignment
	// ErrorCircularImport ...
	ErrorCircularImport
	// ErrorNotAStruct ...
	ErrorNotAStruct
	// ErrorUnknownField ...
	ErrorUnknownField
	// ErrorTemporaryNotAddressable ...
	ErrorTemporaryNotAddressable
	// ErrorContinueOutsideLoop ...
	ErrorContinueOutsideLoop
	// ErrorBreakOutsideLoopOrSwitch ...
	ErrorBreakOutsideLoopOrSwitch
	// ErrorDereferencingNullPointer ...
	ErrorDereferencingNullPointer
	// ErrorLiteralDuplicateField ...
	ErrorLiteralDuplicateField
	// ErrorGroupsCannotBeMerged ...
	ErrorGroupsCannotBeMerged
)

// Error ...
type Error struct {
	code     ErrorCode
	location LocationRange
	args     []string
}

// NewError ...
func NewError(code ErrorCode, loc LocationRange, args ...string) *Error {
	return &Error{code: code, location: loc, args: args}
}

// NewErrorLog ...
func NewErrorLog() *ErrorLog {
	return &ErrorLog{}
}

// AddError ...
func (log *ErrorLog) AddError(code ErrorCode, loc LocationRange, args ...string) *Error {
	err := NewError(code, loc, args...)
	log.Errors = append(log.Errors, err)
	return err
}

// ToString ...
func (log *ErrorLog) ToString(l *LocationMap) string {
	str := ""
	for _, e := range log.Errors {
		str += ErrorToString(e, l) + "\n"
	}
	return str
}

// Error ...
func (e *Error) Error() string {
	return e.ToString()
}

// ToString ...
func (e *Error) ToString() string {
	switch e.code {
	case ErrorUnreachable:
		return "Detected nreachable code"
	case ErrorUninitializedVariable:
		return "Variable " + e.args[0] + " is not initialized"
	case ErrorNamedGroupMerge:
		return "Attempt to merge objects of a named group with other non-heap objects"
	case ErrorScopedGroupMerge:
		return "Attempt to merge objects of two non-nested scopes"
	case ErrorIllegalRune:
		return "Illegal rune"
	case ErrorIllegalString:
		return "Illegal string"
	case ErrorIllegalNumber:
		return "Illegal number"
	case ErrorIllegalCharacter:
		return "Illegal character"
	case ErrorExpectedToken:
		str := "`" + e.args[1] + "`"
		if e.args[1] == "\n" || e.args[1] == "\r\n" {
			str = "`end of line`"
		}
		for i := 2; i < len(e.args); i++ {
			if e.args[i] == "\n" || e.args[i] == "\r\n" {
				str += " or " + "`end of line`"
			} else {
				str += " or " + "`" + e.args[i] + "`"
			}
		}
		if e.args[0] == "\n" || e.args[0] == "\r\n" {
			return "Expected " + str + " but got " + "end of line"
		}
		return "Expected " + str + " but got " + "`" + e.args[0] + "`"
	case ErrorUnexpectedEOF:
		return "Unexpected end of file"
	case ErrorUnknownType:
		return "Unknown type " + e.args[0]
	case ErrorUnknownNamespace:
		return "Unknown namespace " + e.args[0]
	case ErrorUnknownIdentifier:
		return "Unknown identifier " + e.args[0]
	case ErrorArraySizeInteger:
		return "Size of array must be an integer"
	case ErrorCyclicTypeDefinition:
		return "Cyclic type definition"
	case ErrorStructBaseType:
		return "Base type of a struct must be another struct"
	case ErrorStructSingleBaseType:
		return "A struct must have only a single base type"
	case ErrorStructDuplicateInterface:
		return "Interface defined twice in struct"
	case ErrorStructDuplicateField:
		return "Field " + e.args[0] + " defined twice in the same struct"
	case ErrorDuplicateTypeName:
		return "Type name " + e.args[0] + " defined twice"
	case ErrorInterfaceBaseType:
		return "Base type of an interface must be another interface"
	case ErrorInterfaceDuplicateInterface:
		return "Interface defined twice in interface"
	case ErrorInterfaceDuplicateFunc:
		return "Function " + e.args[0] + " declared twice in interface"
	case ErrorDuplicateParameter:
		return "Duplicate parameter name " + e.args[0]
	case ErrorUnnamedParameter:
		return "Parameter " + e.args[0] + " must have a name"
	case ErrorDuplicateScopeName:
		return "The name " + e.args[0] + " has already been defined in this scope"
	case ErrorNotANamespace:
		return e.args[0] + " is not a namespace"
	case ErrorTypeCannotHaveFunc:
		return "Function cannot be attached to this type"
	case ErrorNotAGenericType:
		return "Type is not a generic type"
	case ErrorWrongTypeArgumentCount:
		return "Number of type arguments does not match number of type parameters"
	case ErrorMalformedPackagePath:
		return "Package path " + e.args[0] + " is malformed"
	case ErrorPackageNotFound:
		return "Package " + e.args[0] + " not found"
	case ErrorNameNotExported:
		return "The name " + e.args[0] + " is not exported"
	case ErrorTypeCannotBeInstantiated:
		return "The type " + e.args[0] + " cannot be instantiated"
	case ErrorNumberOutOfRange:
		return "The number " + e.args[0] + " is out of range"
	case ErrorComponentTwice:
		return "Two component declarations in the same file are not allowed"
	case ErrorIncompatibleTypes:
		return "The types are incompatible"
	case ErrorIncompatibleTypeForOp:
		return "Incompatible type for operation"
	case ErrorGenericMustBeInstantiated:
		return "Generic type must be instantiated"
	case ErrorNoValueType:
		return e.args[0] + " used as a value type"
	case ErrorTemporaryNotAssignable:
		return "The expression yields a temporary value and is not assignable"
	case ErrorNotMutable:
		return "The expression yields a non mutable value"
	case ErrorVarWithoutType:
		return "Variable has no type"
	case ErrorExpectedVariable:
		return "Expected variable on left side of the assignment"
	case ErrorWrongMutGroupOrder:
		return "Wrong order or duplication of mutable and group definitions"
	case AssignmentValueCountMismatch:
		return "Number of values on the right-hand side of assignment does not match number of variables on the left-hand side"
	case ErrorNoNewVarsInAssignment:
		return "No new variables on left-hand side of assignment"
	case ErrorCircularImport:
		return "Circular import of package " + e.args[0]
	case ErrorNotAStruct:
		return "The type of the expression is not a struct"
	case ErrorUnknownField:
		return "The field " + e.args[0] + " does not exist"
	case ErrorTemporaryNotAddressable:
		return "The expression yields a temporary value and is not addressable"
	case ErrorContinueOutsideLoop:
		return "`continue` must only be used inside a for statement"
	case ErrorBreakOutsideLoopOrSwitch:
		return "`break` must only be used inside a for or switch statement"
	case ErrorDereferencingNullPointer:
		return "Dereferncing a null pointer"
	case ErrorLiteralDuplicateField:
		return "The field " + e.args[0] + " appears twice in the literal"
	case ErrorGroupsCannotBeMerged:
		var explain string
		i := 0
		if e.args[i] == "both_named" {
			explain = ", because the two named groups `" + e.args[i+1] + "` and `" + e.args[i+2] + "` must not be merged"
			i += 3
		} else if e.args[i] == "both_scoped" {
			explain = ", because both groups belong to a lexical scope and these scopes are not nested"
			i++
		} else if e.args[i] == "scoped_and_named" {
			explain = ", because one group belongs to a lexical scope and these other is the named group `" + e.args[i+1] + "`"
			i += 2
		} else if e.args[i] == "overconstrained" {
			explain = ", because at least one of the groups cannot be statically analyzed (i.e. depends on the control flow) and the other group is not free"
			i++
		} else {
			panic("Oooops")
		}
		if len(e.args) > i {
			return "The expression tries to merge two memory groups (one of them variable `" + e.args[i] + "`) which cannot be merged" + explain
		}
		return "The expression tries to merge two memory groups that cannot be merged" + explain
	}
	panic("Should not happen")
}

// Location ...
func (e *Error) Location() LocationRange {
	return e.location
}

// ErrorToString ...
func ErrorToString(e *Error, l *LocationMap) string {
	loc := e.Location()
	file, line, pos := l.Decode(loc.From)
	//	_, to := l.Resolve(loc.To)
	return fmt.Sprintf("%v %v:%v: %v", file.Name, line, pos, e.ToString())
}
