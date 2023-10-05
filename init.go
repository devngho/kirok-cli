package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var InitCommand = Command{
	name:        "init",
	description: "Initializes a new kirok project.",
	execute: func(args []string) {
		version := "1.1.0"

		if len(args) > 0 {
			version = args[0]
		}

		println("📖 kirok-cli init")
		println("📖 Version: " + version)

		projectName, targetProject, targetWasm, targetBinding := input()

		println()

		targetWasmDir, _ := filepath.Abs(targetWasm)
		targetBindingDir, _ := filepath.Abs(targetBinding)
		targetProjectDir, _ := filepath.Abs(targetProject)

		println()
		println("📖 Check your configuration:")
		println("  Project name: " + projectName)
		fmt.Printf("  Target directory (for project): %s\n", targetProjectDir)
		fmt.Printf("  Target directory (for wasm): %s\n", targetWasmDir)
		fmt.Printf("  Target directory (for binding): %s\n", targetBindingDir)
		println()
		println("Enter to continue.")
		_, _ = fmt.Scanln()

		print("📖 Creating directories...")
		ifErrPanic(os.MkdirAll(targetProjectDir, 0755))
		ifErrPanic(os.MkdirAll(targetWasmDir, 0755))
		ifErrPanic(os.MkdirAll(targetBindingDir, 0755))
		println(" Done!")

		print("📖 Initializing Gradle projects...")
		executable, _ := filepath.Abs(downloadGradle())
		gradleInit(executable, targetProjectDir, projectName)
		println(" Done!")

		print("📖 Initializing kirok...")
		kirokInit(targetProjectDir, targetWasmDir, targetBindingDir, version)
		println(" Done!")

		println("")
		println("🎉  Successfully initialized kirok project!")
		println("📖  What's next:")
		println("  Set java sdk to 19 in IDEA.")
		println("  Then add your bindings in build.gradle.kts.")
		println("  Auto re-build projects with gradle --continuous assemble")
	},
}

func ifErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func unzip(src, dest string) {
	archive, err := zip.OpenReader(src)
	if err != nil {
		panic(err)
	}
	defer func(archive *zip.ReadCloser) {
		_ = archive.Close()
	}(archive)

	for _, f := range archive.File {
		filePath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		_ = dstFile.Close()
		_ = fileInArchive.Close()
	}
}

func downloadGradle() string {
	url := "https://downloads.gradle.org/distributions/gradle-8.3-bin.zip"
	tempFolder := os.TempDir()
	out, _ := os.OpenFile(filepath.Join(tempFolder, "gradle.zip"), os.O_RDONLY, 0644)
	if _, err := out.Stat(); err != nil {
		out, err = os.Create(filepath.Join(tempFolder, "gradle.zip"))

		ifErrPanic(err)
		resp, err := http.Get(url)
		ifErrPanic(err)
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		_, _ = io.Copy(out, resp.Body)

		unzip(filepath.Join(tempFolder, "gradle.zip"), tempFolder)
	}
	return filepath.Join(tempFolder, "gradle-8.3", "bin", "gradle")
}

func input() (string, string, string, string) {
	var projectName string
	var targetProject string
	var targetWasm string
	var targetBinding string

	print("❓  Project name: ")
	_, _ = fmt.Scanln(&projectName)

	print("❓  Target directory (for project) : ")
	_, _ = fmt.Scanln(&targetProject)

	print("❓  Target directory (for wasm) : ")
	_, _ = fmt.Scanln(&targetWasm)

	print("❓  Target directory (for binding) : ")
	_, _ = fmt.Scanln(&targetBinding)

	return projectName, targetProject, targetWasm, targetBinding
}

func gradleInit(executable string, targetProjectDir string, projectName string) {
	cmd := exec.Command(executable, "init", "--type", "basic", "--dsl", "kotlin", "--project-name", projectName)
	cmd.Dir = targetProjectDir
	_ = cmd.Run()
}

func kirokInit(targetProjectDir string, targetWasmDir string, targetBindingDir string, version string) {
	// 1. Modify settings.gradle.kts
	file, _ := os.OpenFile(filepath.Join(targetProjectDir, "settings.gradle.kts"), os.O_APPEND|os.O_WRONLY, 0644)
	defer func() {
		_ = file.Close()
	}()
	_, _ = file.WriteString(
		`
pluginManagement {
    repositories {
        mavenCentral()
        mavenLocal()
        gradlePluginPortal()
    }
}
`)
	// 2. Modify build.gradle.kts
	file, _ = os.OpenFile(filepath.Join(targetProjectDir, "build.gradle.kts"), os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		_ = file.Close()
	}()

	relWasmDir, _ := filepath.Rel(targetProjectDir, targetWasmDir)
	relBindingDir, _ := filepath.Rel(targetProjectDir, targetBindingDir)

	_, _ = file.WriteString(
		fmt.Sprintf(`
import io.github.devngho.kirok.plugin.kirok
import org.jetbrains.kotlin.gradle.targets.js.binaryen.BinaryenRootPlugin

plugins {
    kotlin("multiplatform") version "1.9.0"
    kotlin("plugin.serialization") version "1.9.0"
    id("com.google.devtools.ksp") version "1.9.0-1.0.13"
    id("io.github.devngho.kirok.plugin") version "%s"
}

group = "com.example"
version = "1.0-SNAPSHOT"

repositories {
    mavenCentral()
    mavenLocal()
}

kotlin {
    jvm()
    wasm {
        binaries.executable()
        browser {
			webpackTask {
                enabled = false
            }
		}
        applyBinaryen()
    }
    sourceSets {
        val jvmMain by getting {
			dependencies {
				// Add your bindings, dependencies here
			}
		}
    }
}

kirok {
    wasmDir = "%s"
    wasmJsDir = "%s"
    bindingDir = "%s"
    // Add your bindings here
    binding = listOf()
}

dependencies.kirok(project)
`, filepath.ToSlash(relWasmDir), filepath.ToSlash(relBindingDir), filepath.ToSlash(relBindingDir), version))

	// 3. add src directory
	ifErrPanic(os.MkdirAll(filepath.Join(targetProjectDir, "src", "commonMain", "kotlin"), 0755))

	// 4. add sample code
	file, _ = os.OpenFile(filepath.Join(targetProjectDir, "src", "commonMain", "kotlin", "Sample.kt"), os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		_ = file.Close()
	}()
	_, _ = file.WriteString(
		`
import io.github.devngho.kirok.Init
import io.github.devngho.kirok.Intent
import io.github.devngho.kirok.Model
import kotlinx.serialization.Serializable

@Serializable
@Model
data class Sample(var count: Int)

@Init
fun init(): Sample = Sample(0)

@Intent
fun increment(counter: Sample) {
    counter.count++
}
`)
}
