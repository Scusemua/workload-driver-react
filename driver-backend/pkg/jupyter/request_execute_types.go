package jupyter

import (
	"encoding/json"
	"log"
)

// RequestExecuteArgsAdditionalArguments is a struct that includes non-standard arguments that may be passed to
// the KernelConnection.RequestExecute method (i.e., the RequestExecute method of a KernelConnection interface).
//
// Non-standard arguments are arguments which are not part of the Jupyter RequestKernel interface, and are instead
// specific to this module/library. They serve a purpose that is, for all intents and purposes, only relevant/realized
// by this module/library.
type RequestExecuteArgsAdditionalArguments struct {
	// AwaitResponse is a boolean flag that, if true, indicates that the caller will wait for the response to the
	// "execute_request" message before returning.
	//
	// Default: true
	AwaitResponse bool `json:"await_response"`

	// RequestMetadata is a JSON-serializable mapping that will be included within the "metadata" frame
	// of the "execute_request" message.
	//
	// If anything contained within this RequestMetadata field is not JSON serializable, then an error will
	// occur when the attempt is made to embed the contents of RequestMetadata in the "metadata" frame of
	// the "execute_request" message.
	RequestMetadata map[string]interface{} `json:"request_metadata"`

	// ResponseCallback defines a function that should be called when the result ("execute_reply") is received.
	ResponseCallback func(response KernelMessage) `json:"-"`
}

// RequestExecuteArgs defines the arguments that can be passed to the KernelConnection.RequestExecute method (i.e., the
// RequestExecute method of a KernelConnection interface).
//
// See the [Official Jupyter Kernel Messaging Documentation] for additional information concerning these parameters.
//
// [Official Jupyter Kernel Messaging Documentation]: https://jupyter-client.readthedocs.io/en/latest/messaging.html#execute
type RequestExecuteArgs struct {
	ExtraArguments *RequestExecuteArgsAdditionalArguments `json:"extra_arguments"`

	// Code is the Python code to be executed.
	Code string `json:"code"`

	// Silent is a boolean flag which, if true, signals the kernel to execute the code as "quietly" as possible.
	// Setting Silent to true will necessarily force StoreHistory to be set to false within the Jupyter kernel.
	//
	// Default: false.
	Silent bool `json:"silent"`

	// StoreHistory is a boolean flag which, if true, signals the kernel to populate history.
	//
	// Default: true if Silent is false. If Silent is true, then StoreHistory will be forced to false.
	StoreHistory bool `json:"store_history"`

	// UserExpressions is a mapping of names to expressions to be evaluated in the "user's dict" within the kernel.
	// The rich display-data representation of each will be evaluated after execution.
	//
	// See the [display_data content for the structure of the representation data] for additional information.
	//
	// [display_data content for the structure of the representation data]: https://jupyter-client.readthedocs.io/en/latest/messaging.html#id4
	UserExpressions map[string]interface{} `json:"user_expressions"`

	// AllowStdin is a boolean flag that, when true, enables the kernel to prompt the user for input when executing code.
	// This is performed via an ["input_request" message].
	//
	// If AllowStdin is false, then the kernel will not send these messages.
	//
	// By default, AllowStdin is set to True.
	//
	// ["input_request" message]: https://jupyter-client.readthedocs.io/en/latest/messaging.html#messages-on-the-stdin-router-dealer-channel
	AllowStdin bool `json:"allow_stdin"`

	// StopOnError is a boolean flag, which, if true, aborts the execution queue if an exception is encountered.
	// If StopOnError is set to false, then queued execute_requests will execute even if this request generates an exception.
	//
	// By default, StopOnError is set to True.
	StopOnError bool `json:"stop_on_error"`
}

// AwaitResponse returns the AwaitResponse argument of the target RequestExecuteArgs, which defaults to true.
func (args *RequestExecuteArgs) AwaitResponse() bool {
	if args.ExtraArguments != nil {
		return args.ExtraArguments.AwaitResponse
	}

	return true
}

// RequestMetadata returns the RequestMetadata argument of the target RequestExecuteArgs,
// or nil if there is no request metadata.
func (args *RequestExecuteArgs) RequestMetadata() map[string]interface{} {
	if args.ExtraArguments != nil {
		return args.ExtraArguments.RequestMetadata
	}

	return nil
}

// StripNonstandardArguments returns a copy of the target RequestExecuteArgs with the non-standard arguments stripped
// away so that they are no longer present (and therefore will not be present in a JSON serialization of the struct).
func (args *RequestExecuteArgs) StripNonstandardArguments() *RequestExecuteArgs {
	// Create a copy of the target RequestExecuteArgs struct.
	argsCopy := *args

	// Strip out the ExtraArguments, if they exist. If they don't, then this is a no-op, of course.
	argsCopy.ExtraArguments = nil

	// Return the copy, which has had the ExtraArguments stripped out.
	return &argsCopy
}

// String returns a string representation of the target RequestExecuteArgs struct suitable for logging.
//
// This method ultimately marshals the struct to JSON and thus will panic if the JSON serialization fails.
// This is only likely to happen if there is some non-JSON-serializable entry in the RequestExecuteArgs struct's
// RequestMetadata field.
func (args *RequestExecuteArgs) String() string {
	m, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}

	return string(m)
}

// RequestExecuteArgsBuilder is a struct that exists to facilitate the creation of RequestExecuteArgs structs.
type RequestExecuteArgsBuilder struct {
	args RequestExecuteArgs
}

func NewRequestExecuteArgsBuilder() *RequestExecuteArgsBuilder {
	return &RequestExecuteArgsBuilder{args: RequestExecuteArgs{
		ExtraArguments: &RequestExecuteArgsAdditionalArguments{
			AwaitResponse:   true,
			RequestMetadata: make(map[string]interface{}),
		},
		Silent:       false,
		StoreHistory: false,
	}}
}

func (b *RequestExecuteArgsBuilder) Code(code string) *RequestExecuteArgsBuilder {
	b.args.Code = code
	return b
}

// Silent assigns the value of the Silent parameter.
// Silent is a boolean flag which, if true, signals the kernel to execute the code as "quietly" as possible.
// Note that setting Silent to true will necessarily force StoreHistory to be set to false within the Jupyter kernel.
//
// See the [Official Jupyter Kernel Messaging Documentation] for additional information concerning these parameters.
//
// [Official Jupyter Kernel Messaging Documentation]: https://jupyter-client.readthedocs.io/en/latest/messaging.html#execute
func (b *RequestExecuteArgsBuilder) Silent(silent bool) *RequestExecuteArgsBuilder {
	b.args.Silent = silent

	// If Silent is true, then StoreHistory is forced to false.
	//
	// While we don't necessarily have to enforce this here, we're doing so to avoid the warning
	// message from Jupyter when the request is submitted.
	if silent {
		b.args.StoreHistory = false
	}

	return b
}

// StoreHistory sets the value of the StoreHistory parameter, which is a boolean flag that, when true, signals the
// kernel to populate history.
//
// Note: setting the Silent parameter to true will force StoreHistory to be set to false within the Jupyter kernel.
//
// See the [Official Jupyter Kernel Messaging Documentation] for additional information concerning these parameters.
//
// [Official Jupyter Kernel Messaging Documentation]: https://jupyter-client.readthedocs.io/en/latest/messaging.html#execute
func (b *RequestExecuteArgsBuilder) StoreHistory(storeHistory bool) *RequestExecuteArgsBuilder {
	b.args.StoreHistory = storeHistory

	if b.args.Silent && storeHistory {
		log.Printf(
			"[WARNING] The 'Silent' argument of the \"execute_request\" arguments is already set to true. " +
				"As a result, 'StoreHistory' will be forced to false by the Jupyter kernel. " +
				"Setting it to true will not have the intended effect unless 'Silent' is changed to be false.")
	}

	return b
}

func (b *RequestExecuteArgsBuilder) UserExpressions(userExpressions map[string]interface{}) *RequestExecuteArgsBuilder {
	if userExpressions == nil {
		userExpressions = make(map[string]interface{})
	}

	b.args.UserExpressions = userExpressions
	return b
}

func (b *RequestExecuteArgsBuilder) AllowStdin(allowStdin bool) *RequestExecuteArgsBuilder {
	b.args.AllowStdin = allowStdin
	return b
}

func (b *RequestExecuteArgsBuilder) StopOnError(stopOnError bool) *RequestExecuteArgsBuilder {
	b.args.StopOnError = stopOnError
	return b
}

func (b *RequestExecuteArgsBuilder) OnResponseCallback(callback func(response KernelMessage)) *RequestExecuteArgsBuilder {
	if b.args.ExtraArguments == nil {
		b.args.ExtraArguments = &RequestExecuteArgsAdditionalArguments{
			AwaitResponse:    true,
			RequestMetadata:  make(map[string]interface{}),
			ResponseCallback: callback,
		}
	}

	b.args.ExtraArguments.ResponseCallback = callback
	return b
}

func (b *RequestExecuteArgsBuilder) AwaitResponse(awaitResponse bool) *RequestExecuteArgsBuilder {
	if b.args.ExtraArguments == nil {
		b.args.ExtraArguments = &RequestExecuteArgsAdditionalArguments{
			AwaitResponse:   awaitResponse,
			RequestMetadata: make(map[string]interface{}),
		}
	}

	b.args.ExtraArguments.AwaitResponse = awaitResponse

	return b
}

// AddMetadata adds metadata with the given key and value to the RequestExecuteArgs.
//
// If there already exists a metadata entry with the same key, then that entry is silently overwritten.
//
// AddMetadata can be called multiple times to add multiple entries to the request's metadata.
func (b *RequestExecuteArgsBuilder) AddMetadata(key string, value interface{}) *RequestExecuteArgsBuilder {
	if b.args.ExtraArguments == nil {
		b.args.ExtraArguments = &RequestExecuteArgsAdditionalArguments{
			AwaitResponse:   true,
			RequestMetadata: make(map[string]interface{}),
		}
	}

	if b.args.ExtraArguments.RequestMetadata == nil {
		b.args.ExtraArguments.RequestMetadata = make(map[string]interface{})
	}

	b.args.ExtraArguments.RequestMetadata[key] = value

	return b
}

// GetMetadata returns the metadata stored at the given key within the RequestExecuteArgs.
//
// If there is no metadata stored at the given key, then GetMetadata returns nil.
func (b *RequestExecuteArgsBuilder) GetMetadata(key string) interface{} {
	if b.args.ExtraArguments == nil || b.args.ExtraArguments.RequestMetadata == nil {
		return nil
	}

	return b.args.ExtraArguments.RequestMetadata[key]
}

func (b *RequestExecuteArgsBuilder) Build() *RequestExecuteArgs {
	return &b.args
}
