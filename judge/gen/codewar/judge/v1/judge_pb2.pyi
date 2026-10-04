from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class JudgeRequest(_message.Message):
    __slots__ = ("code", "test_cases", "timeout_ms")
    CODE_FIELD_NUMBER: _ClassVar[int]
    TEST_CASES_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    code: str
    test_cases: _containers.RepeatedCompositeFieldContainer[TestCase]
    timeout_ms: int
    def __init__(self, code: _Optional[str] = ..., test_cases: _Optional[_Iterable[_Union[TestCase, _Mapping]]] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class TestCase(_message.Message):
    __slots__ = ("name", "call", "expected")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CALL_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    name: str
    call: str
    expected: str
    def __init__(self, name: _Optional[str] = ..., call: _Optional[str] = ..., expected: _Optional[str] = ...) -> None: ...

class JudgeVerdict(_message.Message):
    __slots__ = ("passed", "category", "duration_ms", "detail")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    category: str
    duration_ms: int
    detail: str
    def __init__(self, passed: _Optional[bool] = ..., category: _Optional[str] = ..., duration_ms: _Optional[int] = ..., detail: _Optional[str] = ...) -> None: ...
