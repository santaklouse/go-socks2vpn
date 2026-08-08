on run
	my launchGUI("")
end run

on open location incomingURL
	my launchGUI(incomingURL)
end open location

on launchGUI(incomingURL)
	set executablePath to POSIX path of (path to resource "socks2vpn-gui")
	set commandText to quoted form of executablePath
	if incomingURL is not "" then
		set commandText to commandText & " --deep-link " & quoted form of incomingURL
	end if
	do shell script commandText with administrator privileges
end launchGUI
