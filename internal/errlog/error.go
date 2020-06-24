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
	// ErrorUnknownLinkage ...
	ErrorUnknownLinkage
	// ErrorNotAFunction ...
	ErrorNotAFunction
	// ErrorParameterCountMismatch ...
	ErrorParameterCountMismatch
	// ErrorIllegalCast ...
	ErrorIllegalCast
	// ErrorNewInitializerMismatch ...
	ErrorNewInitializerMismatch
	// ErrorCannotInferTypeWithGroups ...
	ErrorCannotInferTypeWithGroups
	// ErrorGroupUnavailable ...
	ErrorGroupUnavailable
	// ErrorUnknownMetaProperty ...
	ErrorUnknownMetaProperty
	// ErrorDualOutsideDualFunction ...
	ErrorDualOutsideDualFunction
	// ErrorTargetIsNotMutable ...
	ErrorTargetIsNotMutable
	// ErrorIllegalEllipsis ...
	ErrorIllegalEllipsis
	// ErrorGroupingConstraints ...
	ErrorGroupingConstraints
	// ErrorSliceOfAnonymousArray ...
	ErrorSliceOfAnonymousArray
	// ErrorAddressOfAnonymousValue ...
	ErrorAddressOfAnonymousValue
	// ErrorGroupingOutOfScope ...
	ErrorGroupingOutOfScope
	// ErrorPointerInUnion ...
	ErrorPointerInUnion
	// ErrorExcessiveUnionValue ...
	ErrorExcessiveUnionValue
	// ErrorMalformedPackageConfig ...
	ErrorMalformedPackageConfig
	// ErrorPackageNotForTarget ...
	ErrorPackageNotForTarget
	// ErrorUnknownMetaAttribute ...
	ErrorUnknownMetaAttribute
	// ErrorUnexpectedMetaAttributeParam ...
	ErrorUnexpectedMetaAttributeParam
	// ErrorExportOutsideComponent ...
	ErrorExportOutsideComponent
	// ErrorISRInWrongContext ...
	ErrorISRInWrongContext
	// ErrorConcurrentInWrongContext ...
	ErrorConcurrentInWrongContext
	// ErrorNoMangleInWrongContext ...
	ErrorNoMangleInWrongContext
	// ErrorExportInWrongContext ...
	ErrorExportInWrongContext
	// ErrorElementNotAccessible ...
	ErrorElementNotAccessible
	// ErrorTypeNotAccessible ...
	ErrorTypeNotAccessible
	// ErrorDoubleComponentUsage ...
	ErrorDoubleComponentUsage
	// ErrorCircularComponentUsage ...
	ErrorCircularComponentUsage
	// ErrorWrongTypeForDelete ...
	ErrorWrongTypeForDelete
	// ErrorWrongTypeForDestructor ...
	ErrorWrongTypeForDestructor
	// ErrorInconsistentGroupSpecifier ...
	ErrorInconsistentGroupSpecifier
)

// Error ...
type Error struct {
	code      ErrorCode
	location  LocationRange
	args      []string
	locations []LocationRange
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
	if code == 0 {
		panic("Oooops")
	}
	err := NewError(code, loc, args...)
	log.Errors = append(log.Errors, err)
	return err
}

// AddErrorMulti ...
func (log *ErrorLog) AddErrorMulti(code ErrorCode, loc []LocationRange, args ...string) *Error {
	if code == 0 {
		panic("Oooops")
	}
	err := NewError(code, loc[0], args...)
	err.locations = loc
	log.Errors = append(log.Errors, err)
	return err
}

// AddGroupingError ...
func (log *ErrorLog) AddGroupingError(group1 string, loc1 LocationRange, group2 string, locs ...LocationRange) *Error {
	err := &Error{code: ErrorGroupingConstraints, args: []string{group1, group2}, location: loc1}
	err.locations = append(err.locations, locs...)
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
	return e.ToString(nil)
}

// ToString ...
func (e *Error) ToString(l *LocationMap) string {
	switch e.code {
	case ErrorUnreachable:
		return "Detected unreachable code"
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
		return "Dereferencing a null pointer"
	case ErrorLiteralDuplicateField:
		return "The field " + e.args[0] + " appears twice in the literal"
	case ErrorGroupsCannotBeMerged:
		var explain string
		i := 0
		if e.args[i] == "both_named" {
			if e.args[i+1][0] == '-' && e.args[i+2][0] == '-' {
				explain = ", because the two isolated groups must not be merged"
			} else if e.args[i+1][0] == '-' {
				explain = ", because the isolated group must not be merged with group `" + e.args[i+2] + "`"
			} else if e.args[i+2][0] == '-' {
				explain = ", because the isolated group must not be merged with group `" + e.args[i+1] + "`"
			} else {
				explain = ", because the two named groups `" + e.args[i+1] + "` and `" + e.args[i+2] + "` must not be merged"
			}
			i += 3
		} else if e.args[i] == "both_scoped" {
			explain = ", because both groups belong to a lexical scope and these scopes are not nested"
			i++
		} else if e.args[i] == "scoped_and_named" {
			if e.args[i+1][0] == '-' {
				explain = ", because one group belongs to a lexical scope and the other one is an isolated group"
			} else {
				explain = ", because one group belongs to a lexical scope and these other is the named group `" + e.args[i+1] + "`"
			}
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
	case ErrorUnknownLinkage:
		return "Unknown linkage " + e.args[0]
	case ErrorNotAFunction:
		return "The expression is not a function"
	case ErrorParameterCountMismatch:
		return "Argument count does not match parameter count"
	case ErrorIllegalCast:
		return "The type conversion from " + e.args[0] + " to " + e.args[1] + " is not allowed"
	case ErrorNewInitializerMismatch:
		return "The initializer does not match the data type"
	case ErrorCannotInferTypeWithGroups:
		return "Cannot infer type with group specifiers"
	case ErrorGroupUnavailable:
		return "The memory group is unavailable at this place, most likely because it has been assigned to an isolated pointer before"
	case ErrorUnknownMetaProperty:
		return "Unknown type property `" + e.args[0] + "`"
	case ErrorDualOutsideDualFunction:
		return "`dual` is used outside of a dual function"
	case ErrorTargetIsNotMutable:
		return "The target for the member function is not mutable, but the function requires a mutable target"
	case ErrorIllegalEllipsis:
		return "The `...` operator is not allowed in this context"
	case ErrorGroupingConstraints:
		if l == nil {
			return "The " + e.args[0] + " and the " + e.args[1] + " cannot be merged into one group"
		}
		str := "The " + e.args[0] + " and the " + e.args[1] + " cannot be merged into one group"
		for _, loc := range e.locations {
			_, line, pos := l.Decode(loc.From)
			str += fmt.Sprintf("\n\tvia %v:%v", line, pos)
		}
		return str
	case ErrorInconsistentGroupSpecifier:
		str := "The group specifier named " + e.args[0] + " is different from the one used here:"
		for _, loc := range e.locations[1:] {
			_, line, pos := l.Decode(loc.From)
			str += fmt.Sprintf(" %v:%v", line, pos)
		}
		return str
	case ErrorAddressOfAnonymousValue:
		// This can happen when a pointer is dereferened, and then the address is taken,
		// e.g. `&*arrayPtr`
		return "Taking the address of an anonymous value is not allowed"
	case ErrorSliceOfAnonymousArray:
		// This can happen when a pointer to an array is dereferened, and then a slice is taken,
		// e.g. `(*arrayPtr)[1:2]`
		return "Taking a slice of an anonymous array is not allowed"
	case ErrorGroupingOutOfScope:
		return "The variable uses a grouping that is out of scope. Most likely the grouping contains a pointer to some stack variable and this stack variable is out of scope"
	case ErrorPointerInUnion:
		return "Pointer types must not be used in unions"
	case ErrorExcessiveUnionValue:
		return "Union initializers must not contain values for more than one union field"
	case ErrorMalformedPackageConfig:
		return "Malformed package.json: " + e.args[0]
	case ErrorPackageNotForTarget:
		return "The package cannot be built for the specified target. Inspect package.json for details"
	case ErrorUnknownMetaAttribute:
		return "Unknown meta attribute " + e.args[0]
	case ErrorUnexpectedMetaAttributeParam:
		return "Unexpected parameter for meta attribute " + e.args[0]
	case ErrorExportOutsideComponent:
		return "The [export] attribute can only be used inside of a component context"
	case ErrorISRInWrongContext:
		return "The [isr] attribute must not be used on functions of non-static components"
	case ErrorConcurrentInWrongContext:
		return "The [concurrent] attribute can only be applied on a struct type"
	case ErrorNoMangleInWrongContext:
		return "The [nomangle] attribute cannot be used on this function"
	case ErrorExportInWrongContext:
		return "The [export] attribute cannot be used in this context"
	case ErrorElementNotAccessible:
		return "The element " + e.args[0] + " of " + e.args[1] + " is not accessible in this context"
	case ErrorTypeNotAccessible:
		return "The type " + e.args[0] + " of " + e.args[1] + " is not accessible in this context"
	case ErrorDoubleComponentUsage:
		return "The component " + e.args[0] + " uses the component " + e.args[1] + " twice. Remove the redundant usage"
	case ErrorCircularComponentUsage:
		return "The components " + e.args[0] + " and " + e.args[1] + " have a circular usage"
	case ErrorWrongTypeForDelete:
		return "Calling a destructor on this type is not possible"
	case ErrorWrongTypeForDestructor:
		return "A destructor cannot be attached to this type"
	}
	println(e.code)
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
	return fmt.Sprintf("%v %v:%v: %v", file.Name, line, pos, e.ToString(l))
}
