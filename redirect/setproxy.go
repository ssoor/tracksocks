package redirect

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ssoor/winapi"
)

func ClearIEBrowserSafeTip() (bool, error) {
	var openHKey winapi.HKEY
	if errorCode := winapi.RegOpenKeyEx(winapi.HKEY_CURRENT_USER, "Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings", 0, winapi.KEY_READ|winapi.KEY_WRITE, &openHKey); errorCode != 0 {
		return false, errors.New(fmt.Sprint("Open registry filed:", errorCode))
	}

	var warnValue uint32 = 0

	if errorCode := winapi.RegSetValueEx(openHKey, "WarnOnIntranet", 0, winapi.REG_DWORD, (*byte)(unsafe.Pointer(&warnValue)), 4); errorCode != 0 {
		return false, errors.New(fmt.Sprint("Save registry value filed:", errorCode))
	}
	
	winapi.RegCloseKey(openHKey)

	return false, nil
}

func SetPACProxy(autoConfigURL string) (bool, error) {
	var proxyInternetOptions [5]winapi.INTERNET_PER_CONN_OPTION
	var proxyInternetOptionList winapi.INTERNET_PER_CONN_OPTION_LIST

	proxyInternetOptions[0].Option = winapi.INTERNET_PER_CONN_AUTOCONFIG_URL
	proxyInternetOptions[0].Value = uint64(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(autoConfigURL))))

	proxyInternetOptions[1].Option = winapi.INTERNET_PER_CONN_AUTODISCOVERY_FLAGS
	proxyInternetOptions[1].Value = 0

	proxyInternetOptions[2].Option = winapi.INTERNET_PER_CONN_FLAGS
	proxyInternetOptions[2].Value = winapi.PROXY_TYPE_AUTO_PROXY_URL | winapi.PROXY_TYPE_DIRECT

	proxyInternetOptions[3].Option = winapi.INTERNET_PER_CONN_PROXY_BYPASS
	proxyInternetOptions[3].Value = uint64(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("<local>"))))

	proxyInternetOptions[4].Option = winapi.INTERNET_PER_CONN_PROXY_SERVER
	proxyInternetOptions[4].Value = 0

	proxyInternetOptionList.OptionError = 0
	proxyInternetOptionList.Connection = nil

	proxyInternetOptionList.Size = uint32(unsafe.Sizeof(proxyInternetOptionList))

	proxyInternetOptionList.OptionCount = 5
	proxyInternetOptionList.Options = (*winapi.INTERNET_PER_CONN_OPTION)(unsafe.Pointer(&proxyInternetOptions))

	succ, err := ClearIEBrowserSafeTip()

	if succ = winapi.InternetSetOption(nil, winapi.INTERNET_OPTION_PER_CONNECTION_OPTION, &proxyInternetOptionList, proxyInternetOptionList.Size); false == succ {
		err = errors.New("Change internet option failed.")
	}

	return succ, err
}
