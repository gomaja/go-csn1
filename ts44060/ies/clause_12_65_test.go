package ies

import (
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"strings"
	"testing"
)

// TS 44.060 V19.0.0 §12.65, DOCX paragraphs 25735-25751.
func TestClause1265Vectors(t *testing.T) {
	checkPublicationVectors(t, "12.65", "Dynamic Allocation 3 IE", reflect.TypeFor[DynamicAllocation3IE](), []publicationVector{
		{"c0", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"c0", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"c0", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"c0", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"c4", 8, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null}]},"USF2":null}}`, nil},
		{"c63e", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"c63e", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"c63e", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"c540", 12, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]},"USF2":null}}`, nil},
		{"c63f9f6ba0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"c63f9f6ba0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"c63f9f6ba0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"c8", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"c8", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"c8", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"c8", 6, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"cc", 8, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null}]}}}`, nil},
		{"ce3e00", 17, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, nil},
		{"ce3f00", 18, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2"}},
		{"ce3ff0", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}}]}}}`, nil},
		{"cd40", 12, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]}}}`, nil},
		{"ce3ecf9ae8", 39, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, nil},
		{"ce3f67d6bb00", 42, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2", "USFChoice.USF2.USFList.1.USFGroup.USF2", "USFChoice.USF2.USFList.2.USFGroup.USF2"}},
		{"ce3ffcfbdaef40", 51, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":6}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":5}}]}}}`, nil},
		{"de00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"de00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"de00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"de00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"de80", 11, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null}]},"USF2":null}}`, nil},
		{"dec7c0", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"dec7c0", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"dec7c0", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"dea8", 15, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]},"USF2":null}}`, nil},
		{"dec7f3ed74", 39, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"dec7f3ed74", 39, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"dec7f3ed74", 39, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"df00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"df00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"df00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"df00", 9, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"df80", 11, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null}]}}}`, nil},
		{"dfc7c0", 20, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, nil},
		{"dfc7e0", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2"}},
		{"dfc7fe", 24, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}}]}}}`, nil},
		{"dfa8", 15, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]}}}`, nil},
		{"dfc7d9f35d00", 42, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, nil},
		{"dfc7ecfad760", 45, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2", "USFChoice.USF2.USFList.1.USFGroup.USF2", "USFChoice.USF2.USFList.2.USFGroup.USF2"}},
		{"dfc7ff9f7b5de8", 54, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":null,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":6}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":5}}]}}}`, nil},
		{"fe00", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"fe00", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"fe00", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"fe00", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"fe40", 12, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null}]},"USF2":null}}`, nil},
		{"fe63e0", 20, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"fe63e0", 20, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"fe63e0", 20, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"fe54", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]},"USF2":null}}`, nil},
		{"fe63f9f6ba", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"fe63f9f6ba", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"fe63f9f6ba", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"fe80", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fe80", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fe80", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fe80", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fec0", 12, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null}]}}}`, nil},
		{"fee3e0", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, nil},
		{"fee3f0", 22, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2"}},
		{"fee3ff00", 25, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}}]}}}`, nil},
		{"fed4", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]}}}`, nil},
		{"fee3ecf9ae80", 43, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, nil},
		{"fee3f67d6bb0", 46, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2", "USFChoice.USF2.USFList.1.USFGroup.USF2", "USFChoice.USF2.USFList.2.USFGroup.USF2"}},
		{"fee3ffcfbdaef4", 55, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":null,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":6}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":5}}]}}}`, nil},
		{"ffe0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"ffe0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"ffe0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"ffe0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
		{"ffe8", 15, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null}]},"USF2":null}}`, nil},
		{"ffec7c", 23, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"ffec7c", 23, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"ffec7c", 23, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}}]},"USF2":null}}`, nil},
		{"ffea80", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]},"USF2":null}}`, nil},
		{"ffec7f3ed740", 43, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"ffec7f3ed740", 43, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"ffec7f3ed740", 43, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31}},{"USFGroup":{"USF":1,"GAMMA":30}},{"USFGroup":{"USF":2,"GAMMA":29}}]},"USF2":null}}`, nil},
		{"fff0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fff0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fff0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fff0", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":null}}}`, nil},
		{"fff8", 15, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null}]}}}`, nil},
		{"fffc7c", 24, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, nil},
		{"fffc7e00", 25, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2"}},
		{"fffc7fe0", 28, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}}]}}}`, nil},
		{"fffa80", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":null},{"USFGroup":null},{"USFGroup":null}]}}}`, nil},
		{"fffc7d9f35d0", 46, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, nil},
		{"fffc7ecfad7600", 49, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":null}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":null}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":null}}]}}}`, []string{"USFChoice.USF2.USFList.0.USFGroup.USF2", "USFChoice.USF2.USFList.1.USFGroup.USF2", "USFChoice.USF2.USFList.2.USFGroup.USF2"}},
		{"fffc7ff9f7b5de80", 58, `{"EXTENDEDDYNAMICALLOCATION":1,"USFGRANULARITY":1,"ALPHA":15,"TSC":7,"USFChoice":{"Alternative":1,"GAMMA":null,"USF2":{"USFList":[{"USFGroup":{"USF":0,"GAMMA":31,"USF2":7}},{"USFGroup":{"USF":1,"GAMMA":30,"USF2":6}},{"USFGroup":{"USF":2,"GAMMA":29,"USF2":5}}]}}}`, nil},
		{"00", 6, `{"EXTENDEDDYNAMICALLOCATION":0,"USFGRANULARITY":0,"ALPHA":null,"TSC":null,"USFChoice":{"Alternative":0,"GAMMA":{"USFList":null},"USF2":null}}`, nil},
	})
}

// TS 44.060 V19.0.0 table 12.65.1 terminates both allocation lists with 0.
// Larger synthetic lists exercise the printed grammar independently of radio-resource state.
func TestDynamicAllocation3ListAllocationGrowth(t *testing.T) {
	for mode := 0; mode < 2; mode++ {
		baseAllocations := float64(0)
		for _, count := range []int{8, 64, 512, 4096} {
			entryWidth := 10
			prefix := "00000"
			if mode == 1 {
				entryWidth = 15
				prefix = "00001"
			}
			bits := prefix + strings.Repeat(strings.Repeat("1", entryWidth), count) + "0"
			wire := make([]byte, (len(bits)+7)/8)
			for i, bit := range bits {
				if bit == '1' {
					wire[i/8] |= 1 << uint(7-i%8)
				}
			}
			var decoded runtime.Decoded[DynamicAllocation3IE]
			var err error
			allocations := testing.AllocsPerRun(3, func() { decoded, err = DecodeDynamicAllocation3IE(wire) })
			if err != nil || decoded.BitsConsumed != len(bits) {
				t.Fatalf("mode=%d count=%d consumed=%d: %v", mode, count, decoded.BitsConsumed, err)
			}
			if mode == 0 {
				entries := decoded.Value.USFChoice.GAMMA.USFList
				if len(entries) != count || entries[count-1].USFGroup.USF != 7 || entries[count-1].USFGroup.GAMMA != 31 {
					t.Fatal("BTTI list differs")
				}
			} else {
				entries := decoded.Value.USFChoice.USF2.USFList
				if len(entries) != count || entries[count-1].USFGroup.USF != 7 || entries[count-1].USFGroup.GAMMA != 31 || **entries[count-1].USFGroup.USF2 != 7 {
					t.Fatal("RTTI list differs")
				}
			}
			// Bound each larger sample by a linear multiple of the eight-entry sample.
			if count == 8 {
				baseAllocations = allocations
			}
			if allocations > baseAllocations*float64(count)/8+64 {
				t.Fatalf("mode=%d count=%d allocations=%.0f exceed linear allocation-count bound", mode, count, allocations)
			}
			t.Logf("mode=%d count=%d allocations=%.0f", mode, count, allocations)
		}
	}
}
