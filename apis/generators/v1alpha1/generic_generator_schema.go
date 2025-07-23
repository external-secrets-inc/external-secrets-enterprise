// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1alpha1

import (
	"fmt"
	"sync"
)

var (
	genericBuilder = make(map[string]GenericGenerator)
	genericLock    sync.RWMutex
)

func RegisterGeneric(kind string, genericGenerator GenericGenerator) {
	genericLock.Lock()
	defer genericLock.Unlock()

	if _, exists := genericBuilder[kind]; exists {
		panic(fmt.Sprintf("Generic generator %q already registered", kind))
	}
	genericBuilder[kind] = genericGenerator
}

// ForceRegister adds to the schema, overwriting a generator if
// already registered. Should only be used for testing.
func ForceRegisterGeneric(kind string, genericGenerator GenericGenerator) {
	genericLock.Lock()
	genericBuilder[kind] = genericGenerator
	genericLock.Unlock()
}

func GetGenericByKind(kind string) (GenericGenerator, bool) {
	genericLock.RLock()
	genericGenerator, ok := genericBuilder[kind]
	genericLock.RUnlock()
	return genericGenerator, ok
}

func GetAllGeneric() map[string]GenericGenerator {
	return genericBuilder
}
