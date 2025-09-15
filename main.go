package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
)

func installSteamCmdDarwin() (bool, error) {
	return true, nil
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure the destination directory exists.
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// Iterate through each file/dir within the zip file.
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// If the file is a directory, create it.
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		// Create the directory for the file if it doesn't exist.
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		// Open the file inside the zip archive.
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// Create the destination file.
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		// Copy the file content.
		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func untar(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func installSteamCmdWindows(pwdDir string) (bool, error) {
	steamcmdUrl := "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	steamCmdPath := filepath.Join(pwdDir, "steamcmd")
	if err := os.MkdirAll(steamCmdPath, 0777); err != nil {
		return false, err
	}

	downloadFile(steamcmdUrl, filepath.Join(steamCmdPath, "steamcmd.zip"))
	unzip(filepath.Join(steamCmdPath, "steamcmd.zip"), steamCmdPath)
	return true, nil
}

func installSteamCmdLinux(pwdDir string) (bool, error) {
	steamcmdUrl := "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"
	steamCmdPath := filepath.Join(pwdDir, "steamcmd")
	if err := os.MkdirAll(steamCmdPath, 0777); err != nil {
		return false, err
	}

	downloadFile(steamcmdUrl, filepath.Join(steamCmdPath, "steamcmd_linux.tar.gz"))
	untar(filepath.Join(steamCmdPath, "steamcmd_linux.tar.gz"), steamCmdPath)
	return true, nil
}

func getLineByLine(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	buf := make([]byte, 1024)
	var line string
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		line += string(buf[:n])
		if strings.Contains(line, "\n") {
			parts := strings.Split(line, "\n")
			lines = append(lines, parts[:len(parts)-1]...)
			line = parts[len(parts)-1]
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines, nil
}

func steamCmdInstallModMacos(pwd string, modId string) {
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("steamcmd +force_install_dir %s +login anonymous +workshop_download_item 108600 %s +quit", filepath.Join(pwd, "mods"), modId))
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return
	}
}

func steamCmdInstallModLinux(pwd string, modId string) {
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s +force_install_dir %s +login anonymous +workshop_download_item 108600 %s +quit", filepath.Join(pwd, "steamcmd", "steamcmd.sh"), filepath.Join(pwd, "mods"), modId))
	// fmt.Println(cmd.String())
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return
	}
}

func steamCmdInstallModWindows(pwd string, modId string) {
	cmd := exec.Command(fmt.Sprintf("%s +force_install_dir %s +login anonymous +workshop_download_item 108600 %s +quit", filepath.Join(pwd, "steamcmd", "steamcmd.exe"), filepath.Join(pwd, "mods"), modId))
	// fmt.Println(cmd.String())
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return
	}
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	// fmt.Println(exPath)
	// fmt.Println(runtime.GOOS)

	// install steamcmd step

	// installSteamCmdLinux(exPath)
	if runtime.GOOS == "darwin" {

	}
	if runtime.GOOS == "windows" {
		installSteamCmdWindows(exPath)
	}
	if runtime.GOOS == "linux" {
		installSteamCmdLinux(exPath)
	}

	pattern := filepath.Join(exPath, "*.txt")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Println("Error finding files:", err)
		return
	}
	var fileList []string
	for _, match := range matches {
		fileName := strings.Split(match, "/")[len(strings.Split(match, "/"))-1]
		fileList = append(fileList, fileName)
		// fmt.Println("Found file:", fileName)
	}
	// fmt.Println("Found files:", fileList)
	prompt := promptui.Select{
		Label: "Choose mods id text file",
		Items: fileList,
	}

	_, fileChoose, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	// fmt.Printf("You choose %q\n", fileChoose)
	modsId, err := getLineByLine(filepath.Join(exPath, fileChoose))
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	// fmt.Println(modsId)
	for _, mod := range modsId {
		fmt.Println(mod)
		if runtime.GOOS == "darwin" {
			steamCmdInstallModMacos(exPath, mod)
		}
		if runtime.GOOS == "windows" {
			steamCmdInstallModWindows(exPath, mod)
		}
		if runtime.GOOS == "linux" {
			steamCmdInstallModLinux(exPath, mod)
		}
	}

	modsFolder := filepath.Join(exPath, "mods", "steamapps", "workshop", "content", "108600")
	// fmt.Println("Mod folder:", modsFolder)
	folders, err := os.ReadDir(modsFolder)
	if err != nil {
		fmt.Println("Error reading mod folder:", err)
		return
	}
	var folderMods []string
	for _, entity := range folders {
		if entity.IsDir() {
			folderMods = append(folderMods, entity.Name())
		}
	}

	// copy mods to zomboid folder
	homeUserDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory:", err)
		return
	}

	// fmt.Println("User home directory:", homeUserDir)
	zomboidModDir := filepath.Join(homeUserDir, "Zomboid", "mods")
	// fmt.Println("Zomboid mod directory:", zomboidModDir)

	// copy all mods to zomboid mod folder
	for _, folder := range folderMods {
		listModSubFolder := filepath.Join(modsFolder, folder, "mods")
		subFolders, err := os.ReadDir(listModSubFolder)
		if err != nil {
			fmt.Println("Error reading sub mod folder:", err)
			return
		}
		// fmt.Println("Sub folders: ", subFolders)
		for _, subFolder := range subFolders {
			if !subFolder.IsDir() {
				continue
			}

			modSubFolder := filepath.Join(listModSubFolder, subFolder.Name())
			cmd := exec.Command("cp", "-R", modSubFolder, zomboidModDir)
			_, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Error copying mod %s: %v\n", subFolder.Name(), err)
				return
			}

		}
	}
	// fmt.Println(zomboidModDir)

}
