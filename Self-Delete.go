package main

import (
  "bufio"
  "fmt"
  "math/rand"
  "os"
  "time"
  "unicode/utf16"
  "unsafe"

  "golang.org/x/sys/windows"
)

var (
  ntdll              = windows.NewLazySystemDLL("ntdll")
  RtlCopyMemory      = ntdll.NewProc("RtlCopyMemory")
  k32                = windows.NewLazySystemDLL("kernel32")
  GetModuleFileNameW = k32.NewProc("GetModuleFileNameW")
)

func UintPtrToString(cs *uint16) (s string) {
  if cs != nil {
    us := make([]uint16, 0, 256)
    for p := uintptr(unsafe.Pointer(cs)); ; p += 2 {
      u := *(*uint16)(unsafe.Pointer(p))
      if u == 0 {
        return string(utf16.Decode(us))
      }
      us = append(us, u)
    }
  }
  return ""
}

func GetRandomString(n int) string {
  rand.NewSource(time.Now().UnixNano())
  str := "abcdefghijklmnopqrstuvwxyz"
  bytes := []byte(str)
  var result []byte
  for i := 0; i < n; i++ {
    result = append(result, bytes[rand.Intn(len(bytes))])
  }
  return string(result)
}

func DeleteHandle(Handle windows.Handle) {
  type FILE_DISPOSITION_INFO struct {
    DeleteFile uint32
  }

  var FDelete FILE_DISPOSITION_INFO
  iosb := windows.IO_STATUS_BLOCK{}

  BuffSize := int(unsafe.Sizeof(FDelete.DeleteFile))
  buf := make([]byte, int(BuffSize))

  FDeletePtr := (*FILE_DISPOSITION_INFO)(unsafe.Pointer(&buf[0]))
  FDeletePtr.DeleteFile = 1

  err := windows.NtSetInformationFile(Handle, &iosb, &buf[0], uint32(unsafe.Sizeof(FDelete)), windows.FileDispositionInformation)
  if err != nil {
    panic(err)
  }
}

func main() {

  Path := make([]uint16, windows.MAX_PATH)
  GetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&Path[0])), uintptr(windows.MAX_PATH))
  handle := OpenHandleNT(&Path[0])
  RenameHandleNT(handle)
  
  GetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&Path[0])), uintptr(windows.MAX_PATH))
  handle = OpenHandleNT(&Path[0])
  DeleteHandle(handle)
  windows.CloseHandle(handle)
  fmt.Println("Auto Borrando listo")

  fmt.Print("Presiona enter para finalizar")
  bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func OpenHandleNT(Path *uint16) (handle windows.Handle) {
  var allocationSize int64

  NTUnicodeString, err := windows.NewNTUnicodeString("\\??\\" + UintPtrToString(Path))
  if err != nil {
    panic(err)
  }
  oa := &windows.OBJECT_ATTRIBUTES{
    ObjectName: NTUnicodeString,
  }
  oa = &windows.OBJECT_ATTRIBUTES{
    Length:     uint32(unsafe.Sizeof(*oa)),
    ObjectName: oa.ObjectName,
    Attributes: windows.OBJ_CASE_INSENSITIVE,
  }

  iosb := windows.IO_STATUS_BLOCK{}
  err = windows.NtCreateFile(&handle,
    uint32(windows.DELETE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE),
    oa,
    &iosb,
    &allocationSize, uint32(windows.FILE_ATTRIBUTE_NORMAL), 0, uint32(windows.FILE_OPEN), uint32(windows.FILE_NON_DIRECTORY_FILE|windows.FILE_SYNCHRONOUS_IO_NONALERT), uintptr(0), 0)
  if err != nil {
    panic(err)
  }

  return
}

func RenameHandleNT(Handle windows.Handle) {
  DS_STREAM_RENAME := ":" + GetRandomString(6)

  type FILE_RENAME_INFO struct {
    ReplaceIfExists uint32
    RootDirectory   windows.Handle
    FileNameLength  uint32
    FileName        [1]uint16
  }

  var FRename FILE_RENAME_INFO
  Stream, _ := windows.UTF16FromString(DS_STREAM_RENAME)
  size := len(Stream)*2 - 2
  BuffSize := int(unsafe.Offsetof(FRename.FileName)) + size

  iosb := windows.IO_STATUS_BLOCK{}
  buf := make([]byte, int(BuffSize))

  FRenamePtr := (*FILE_RENAME_INFO)(unsafe.Pointer(&buf[0]))
  FRenamePtr.ReplaceIfExists = windows.FILE_RENAME_REPLACE_IF_EXISTS | windows.FILE_RENAME_POSIX_SEMANTICS
  FRenamePtr.FileNameLength = uint32(len(Stream)*2 - 2)
  RtlCopyMemory.Call(uintptr(unsafe.Pointer(&FRenamePtr.FileName)), uintptr(unsafe.Pointer(&Stream[0])), unsafe.Sizeof(Stream))
  err := windows.NtSetInformationFile(Handle, &iosb, &buf[0], uint32(unsafe.Sizeof(FRename)+unsafe.Sizeof(Stream)), windows.FileRenameInformation)
  if err != nil {
    panic(err)
  }
}
