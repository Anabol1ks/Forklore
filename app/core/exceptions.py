class PipelineError(Exception):
    """Base class for pipeline errors."""

class EmptyNoteError(PipelineError):
    pass

class NoteTooShortError(PipelineError):
    pass

class InvalidModeError(PipelineError):
    pass

class ValidationError(PipelineError):
    pass
