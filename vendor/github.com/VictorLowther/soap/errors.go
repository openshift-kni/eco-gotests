package soap

/*
Copyright 2015 Victor Lowther <victor.lowther@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

const (
	BadEnvelope         = "Invalid SOAP: Root should be an Envelope"
	BadTag              = "Invalid SOAP: Unexpected tag in Envelope"
	EnvelopeOverstuffed = "Invalid SOAP: Envelope must have at most 2 children"
	NoEnvelope          = "Invalid SOAP: Document does not have a root element"
	TooManyBody         = "Invalid SOAP: More than one Body"
	TooManyHeader       = "Invalid SOAP: More than one Header"
)
