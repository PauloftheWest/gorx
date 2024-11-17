package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
    "gorx/pkg"
)

/*
#cgo CFLAGS: -I ext/orx/code/include
#cgo LDFLAGS: -lorxd -L ext/orx/code/lib/dynamic/

#include "orx.h"
#include "object/orxObject.h"
#include <stdlib.h>
#include <stdio.h>

orxSTATUS orxFASTCALL Init();
orxSTATUS orxFASTCALL Run();
orxSTATUS orxFASTCALL Exit();
orxSTATUS orxFASTCALL Bootstrap();

typedef const orxCLOCK_INFO cci;

void orxFASTCALL Update(const orxCLOCK_INFO *_pstClockInfo, void *_pContext);

// TODO: How do I enable this?
//__declspec(dllexport) unsigned long NvOptimusEnablement        = 1;
//__declspec(dllexport) int AmdPowerXpressRequestHighPerformance = 1;

static void* allocArgv(unsigned int argc) {
   return malloc(sizeof(char *) * argc);
}
*/
import "C"

func write_header(f *os.File) {
	f.WriteString("package gorx\n\n")
	f.WriteString("/*\n")
	f.WriteString("#cgo CFLAGS: -I ../ext/orx/code/include\n")
	f.WriteString("#cgo LDFLAGS: -lorxd -L ../ext/orx/code/lib/dynamic/\n")

	f.WriteString("#include \"orx.h\"\n")
	f.WriteString("#include \"object/orxObject.h\"\n")
	f.WriteString("#include <stdlib.h>\n")
	f.WriteString("#include <stdio.h>\n")
	f.WriteString("*/\n")
	f.WriteString("import \"C\"\n")
	f.WriteString("import \"unsafe\"\n\n")
}

func generate_struct(structs []string) string {
	defs := ""
	for i := 0; i < len(structs); i++ {
		s := structs[i]
		defs += fmt.Sprintf("type %s struct {\n    %s *C.orx%s\n}\n\n", s, s, strings.ToUpper(s))
	}
	return defs
}

/*
 */

//extern orxDLLAPI void orxFASTCALL           orxObject_Enable(orxOBJECT *_pstObject, orxBOOL _bEnable);

func wrap_parameter(arg_type string, name_type string) string {
    arg := ""
    if arg_type == "void*" {
        arg += "unsafe.Pointer("
        arg += name_type
        arg += ")"
    } else if arg_type == "const orxSTRING*" || arg_type == "orxSTRING*"{
        arg += "(**C.orxCHAR)("
        arg += name_type
        arg += ")"
    } else {
        arg += name_type
    }
    return arg
}

type extern_info struct {
	return_type   string
	function_name string
	arg_type      []string
	name_type     []string
}

func generate_extern(line string) string {
	words := strings.Fields(line)
	pos := 1
	ei := &extern_info{}

    // Stip out unecessary tokens
	for len(words) > pos && (words[pos] == "orxDLLAPI" || words[pos] == "orxFASTCALL" || words[pos] == "const") {
		pos = pos + 1
	}

	if len(words) <= pos {
		return ""
	}

    // Get return type
	ei.return_type = words[pos]
	pos = pos + 1
	if len(words) <= pos {
		return ""
	}

    // pointers in go are in front, wrap around
	for words[pos][0] == '*' {
		ei.return_type += "*"
		words[pos] = words[pos][1:]
	}

    // Stip out unecessary tokens
	for len(words) > pos && (words[pos] == "orxDLLAPI" || words[pos] == "orxFASTCALL" || words[pos] == "const") {
		pos = pos + 1
	}

	if len(words) <= pos {
		return ""
	}

	header := strings.Split(words[pos], "(")
	ei.function_name = header[0]
	arg := ""

	if len(header) > 1 {
		arg += header[1]
	}

	pos = pos + 1

	for len(words) > pos {
		if strings.Contains(words[pos], ",") == true || strings.Contains(words[pos], ")") {
			for words[pos][0] == '*' {
				arg += "*"
				words[pos] = words[pos][1:]
			}

			name := strings.Split(words[pos], ")")[0]

			ei.arg_type = append(ei.arg_type, strings.TrimSpace(arg))
			ei.name_type = append(ei.name_type, strings.Trim(strings.TrimSpace(name), ","))

			arg = ""
			pos = pos + 1
			continue
		}

		arg += " " + words[pos]
		pos = pos + 1
	}

	/*
		fmt.Printf("=======================\n")
		fmt.Printf("ret: %s\n", ei.return_type)
		fmt.Printf("function_name: %s\n", ei.function_name)
		for pos, _ = range ei.arg_type {
			fmt.Printf("%d] arg: '%s' name: '%s'\n", pos, ei.arg_type[pos], ei.name_type[pos])
		}
	*/

    // Function start
	fh := "func "

	arg_pos := 0
	self_func := false
	self_type := ""
	if len(ei.arg_type) > 0 && strings.Contains(ei.arg_type[0], "*") {
		toks := strings.Split(ei.arg_type[0], " ")
		name := toks[len(toks)-1]
		name = name[0 : len(name)-1]
		name = strings.Trim(name, "orx")
		name = strings.ToLower(name)
		self_type = name
		fh += "(self *" + name + ") "
		arg_pos++
		self_func = true
	}

	if self_func {
		fn := strings.Split(ei.function_name, "_")
		fh += fn[len(fn)-1] + " ("
	} else {
		fh += strings.Trim(ei.function_name, "orx") + " ("
	}

    // Function Arguments
	first_arg := true
	for arg_pos < len(ei.arg_type) {
		if !first_arg {
			fh += ", "
		}

		// Add name
		fh += ei.name_type[arg_pos] + " "

		// Now add the arg, wrapping around the *
		arg = ei.arg_type[arg_pos]
		for arg[len(arg)-1] == '*' {
			fh += "*"
			arg = arg[0 : len(arg)-1]
		}

        // Go ignores const
        arg = strings.ReplaceAll(arg, "const ", "")

		fh += "C." + arg

		first_arg = false
		arg_pos++
	}

	fh += " ) "
	rt := ei.return_type

	for ei.return_type != "void*" && rt[len(rt)-1] == '*' {
		fh += "*"
		rt = rt[0 : len(rt)-1]
	}

    if ei.return_type == "void*" {
        rt = "string"
    }

    if ei.return_type != "void" {
        if rt != "string" {
            fh += "C."
        }
	    fh += rt
    }

	fh += " {\n"
	fh += "    "

    // Now the return statement
    if ei.return_type != "void" {
        fh += "return "
    }

    if ei.return_type == "void*" {
        fh += "C.GoString((*C.char)(*(*unsafe.Pointer)("
      //return C.GoString((*C.char)(*(*unsafe.Pointer)(C.orxObject_GetUserData(self.object))))
    }

    fh += "C." + ei.function_name + "("
	need_comma := false
	arg_pos = 0

	if self_func {
		fh += "self." + self_type
		need_comma = true
		arg_pos++
	}
	for arg_pos < len(ei.arg_type) {
		if need_comma {
			fh += ", "
		}
        fh += wrap_parameter(ei.arg_type[arg_pos], ei.name_type[arg_pos])
		arg_pos++
		need_comma = true
	}

	fh += ")"
    if ei.return_type == "void*" {
        fh += ")))"
    }
    fh += "\n}\n\n"

	return fh
}

func main() {
	//C.orx_Execute(argc, argv, GORXInit, GORXRun, GORXExit);
	argv := os.Args
	var argc C.uint = C.uint(len(os.Args))
	c_argv := (*[0xfff]*C.char)(C.allocArgv(argc))
	defer C.free(unsafe.Pointer(c_argv))

	for i, arg := range argv {
		c_argv[i] = C.CString(arg)
		defer C.free(unsafe.Pointer(c_argv[i]))
	}

	gorx.Object_Create()
	// Set the bootstrap function to provide at least one resource storage before loading any config files
	C.orxConfig_SetBootstrap(C.orxCONFIG_BOOTSTRAP_FUNCTION(C.Bootstrap))

	fmt.Println("Hello, World!")

	file, err := os.Open("./ext/orx/code/include/object/orxObject.h")
	if err != nil {
		fmt.Println("opening file error", err)
	}
	defer file.Close()

	f, e := os.Create("pkg/obj.go")
	if e != nil {
		panic(e)
	}
	write_header(f)
	f.WriteString(generate_struct([]string{"object", "vector", "obox"}))
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "extern") == true {
			f.WriteString(generate_extern(line))
		}
	}
	//NewObject()
	//scanner := bufio.NewScanner("./ext/orx/code/include/object/orxObject.h")
	//scanner.Split(bufio.ScanLines)
	//gorx.NewObject()

	//gorx.Create()
	/*
		C.orx_Execute(
			argc,
			(**C.char)(unsafe.Pointer(c_argv)),
			C.orxMODULE_INIT_FUNCTION(C.Init),
			C.orxMODULE_RUN_FUNCTION(C.Run),
			C.orxMODULE_EXIT_FUNCTION(C.Exit))
	*/
}

//export Bootstrap
func Bootstrap() C.orxSTATUS {

	//orxResource_AddStorage(orxCONFIG_KZ_RESOURCE_GROUP, "../data/config", orxFALSE);
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)
	C.orxConfig_Load(C.CString("/home/pwest/repos/gorx/test.ini"))
	//return C.orxSTATUS_SUCCESS;
	// Return orxSTATUS_FAILURE to prevent orx from loading the default config file
	return C.orxSTATUS_FAILURE
}

//export Init
func Init() C.orxSTATUS {
	fmt.Printf("INIT!\n")

	// Register the Update function to the core clock
	C.orxClock_Register(
		C.orxClock_Get(C.CString(C.orxCLOCK_KZ_CORE)),
		C.orxCLOCK_FUNCTION(C.Update),
		C.NULL,
		C.orxMODULE_ID_MAIN,
		C.orxCLOCK_PRIORITY_NORMAL)

	return C.orxSTATUS_SUCCESS
}

// typedef void (orxFASTCALL *orxCLOCK_FUNCTION)(const orxCLOCK_INFO *_pstClockInfo, void *_pContext);
// func Update(clockinfo *C.orxCLOCK_INFO, pcontext *C.void) {
//
//export Update
func Update(clockinfo *C.cci, pcontext *C.void) {
	//fmt.Printf("UPDATE!\n");
	// Should quit?
	if C.orxInput_HasBeenActivated(C.CString("Quit")) == C.orxTRUE {
		fmt.Printf("QUIT!!!!\n")
		// Send close event
		C.orxEvent_SendShort(C.orxEVENT_TYPE_SYSTEM, C.orxSYSTEM_EVENT_CLOSE)
	}
}

//export Run
func Run() C.orxSTATUS {

	return C.orxSTATUS_SUCCESS
}

//export Exit
func Exit() C.orxSTATUS {
	fmt.Println("RUN!\n")

	return C.orxSTATUS_SUCCESS
}
