#Requires AutoHotkey v2.0
#SingleInstance Force

global AIW_SAY_EXE := A_ScriptDir "\aiw-say.exe"
global busy := false

^!j::TranslateClipboard("ja")
^!e::TranslateClipboard("en")
^!b::TranslateClipboard("ja", "ja-business")
^!t::OpenTranslationDialog()

TranslateClipboard(target, profile := "") {
	global AIW_SAY_EXE, busy
	if busy {
		MsgBox("A translation is already running.", "AIW Say")
		return
	}
	busy := true
	try {
		if !ClipWait(1) {
			MsgBox("The clipboard does not contain text.", "AIW Say")
			return
		}
		original := A_Clipboard
		if Trim(original) = "" {
			MsgBox("The clipboard text is empty.", "AIW Say")
			return
		}
		try {
			result := RunTranslator(original, target, profile)
		} catch as err {
			MsgBox(err.Message, "AIW Say")
			return
		}
		if result.exitCode != 0 {
			message := result.stderr != "" ? result.stderr : "aiw-say exited with code " result.exitCode
			MsgBox(message, "AIW Say")
			return
		}
		if result.stdout = "" {
			MsgBox("aiw-say returned an empty translation.", "AIW Say")
			return
		}
		if A_Clipboard !== original {
			MsgBox("The clipboard changed while translating. The new clipboard content was kept.", "AIW Say")
			return
		}
		A_Clipboard := result.stdout
	} finally {
		busy := false
	}
}

RunTranslator(text, target, profile) {
	global AIW_SAY_EXE
	if !FileExist(AIW_SAY_EXE)
		throw Error("aiw-say.exe was not found beside this script: " AIW_SAY_EXE)
	command := QuoteArg("powershell.exe") " -NoLogo -NoProfile -NonInteractive -File "
		. QuoteArg(A_ScriptDir "\aiw-say-hotkeys.ps1") " " QuoteArg(AIW_SAY_EXE) " " QuoteArg(target)
	if profile != ""
		command .= " " QuoteArg(profile)
	try process := ComObject("WScript.Shell").Exec(command)
	catch as err
		throw Error("Could not start the translation helper: " err.Message)
	process.StdIn.Write(Utf8Base64Encode(text) "`n")
	process.StdIn.Close()
	protocol := process.StdOut.ReadAll()
	processError := process.StdErr.ReadAll()
	if process.ExitCode != 0
		throw Error(processError != "" ? processError : "The translation helper failed.")
	fields := StrSplit(Trim(protocol, "`r`n"), "|")
	if fields.Length != 3
		throw Error("The translation helper returned an invalid response.")
	stdout := Utf8Base64Decode(fields[2])
	if SubStr(stdout, -1) = "`n"
		stdout := SubStr(stdout, 1, StrLen(stdout) - 1)
	if SubStr(stdout, -1) = "`r"
		stdout := SubStr(stdout, 1, StrLen(stdout) - 1)
	return {
		exitCode: Integer(fields[1]),
		stdout: stdout,
		stderr: Utf8Base64Decode(fields[3])
	}
}

OpenTranslationDialog() {
	global AIW_SAY_EXE
	if !FileExist(AIW_SAY_EXE) {
		MsgBox("aiw-say.exe was not found beside this script: " AIW_SAY_EXE, "AIW Say")
		return
	}
	try exitCode := RunWait(QuoteArg(AIW_SAY_EXE) " --dialog zenity",, "Hide")
	catch as err {
		MsgBox(err.Message, "AIW Say")
		return
	}
	if exitCode != 0
		MsgBox("The Zenity translation dialog failed. Check that native Windows Zenity is installed.", "AIW Say")
}

QuoteArg(value) {
	return Chr(34) . StrReplace(value, Chr(34), Chr(92) . Chr(34)) . Chr(34)
}

Utf8Base64Encode(text) {
	byteCount := StrPut(text, "UTF-8") - 1
	bytes := Buffer(byteCount, 0)
	StrPut(text, bytes, "UTF-8")
	capacity := 4 * Ceil(byteCount / 3) + 8
	encoded := Buffer(capacity * 2, 0)
	encodedChars := capacity
	if !DllCall("Crypt32\CryptBinaryToStringW", "Ptr", bytes, "UInt", byteCount, "UInt", 0x40000001, "Ptr", encoded, "UIntP", &encodedChars)
		throw Error("Could not encode clipboard text for the helper process.")
	return StrGet(encoded, "UTF-16")
}

Utf8Base64Decode(value) {
	if value = "-"
		return ""
	byteCapacity := Ceil(StrLen(value) * 3 / 4) + 4
	bytes := Buffer(byteCapacity, 0)
	byteCount := byteCapacity
	if !DllCall("Crypt32\CryptStringToBinaryW", "Str", value, "UInt", 0, "UInt", 1, "Ptr", bytes, "UIntP", &byteCount, "Ptr", 0, "Ptr", 0)
		throw Error("Could not decode the helper response.")
	return StrGet(bytes, byteCount, "UTF-8")
}
