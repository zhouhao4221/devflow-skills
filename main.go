package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed plugins
var skillsFS embed.FS

var tools = map[string]bool{
	"opencode": true,
	"claude":   true,
	"codex":    true,
}

type skillInfo struct {
	Plugin  string
	Name    string
	Desc    string
	RawPath string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "install":
		runInstall(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	case "uninstall":
		runUninstall(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`devflow-skills — AI 技能一键安装/管理工具

用法:
  devflow-skills install  --tool <TOOL> [--skill <NAME>...] [--all] [--dir <PATH>]
  devflow-skills list     [--plugin <NAME>] [--format text|json]
  devflow-skills uninstall --tool <TOOL> [--skill <NAME>...] [--all] [--dir <PATH>]

命令:
  install     安装技能到目标 AI 工具目录
  list        列出所有可用技能
  uninstall   从目标 AI 工具目录卸载技能

支持的 --tool 值: opencode, claude, codex

示例:
  npx devflow-skills install --tool opencode --all
  npx devflow-skills install --tool opencode --skill req-dev --skill req-review
  npx devflow-skills list --plugin req
  npx devflow-skills uninstall --tool opencode --all
`)
}

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	tool := fs.String("tool", "", "目标 AI 工具: opencode / claude / codex")
	allFlag := fs.Bool("all", false, "安装所有技能")
	dir := fs.String("dir", ".", "目标项目根目录")
	var skills skillsFlag
	fs.Var(&skills, "skill", "要安装的技能名")

	fs.Parse(args)

	if *tool == "" {
		fmt.Fprintln(os.Stderr, "错误: --tool 是必需参数")
		os.Exit(1)
	}
	if !tools[*tool] {
		fmt.Fprintf(os.Stderr, "错误: 不支持的 --tool 值: %s (支持: opencode, claude, codex)\n", *tool)
		os.Exit(1)
	}
	if !*allFlag && len(skills) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 请指定 --skill 或 --all")
		os.Exit(1)
	}
	if *allFlag && len(skills) > 0 {
		fmt.Fprintln(os.Stderr, "错误: --skill 与 --all 不能同时使用")
		os.Exit(1)
	}

	allSkills, err := loadAllSkills()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载技能失败: %v\n", err)
		os.Exit(1)
	}

	var selected []skillInfo
	if *allFlag {
		selected = allSkills
	} else {
		for _, name := range skills {
			matched := resolveSkill(name, allSkills)
			if len(matched) == 0 {
				fmt.Fprintf(os.Stderr, "错误: 未找到技能 '%s'\n", name)
				fmt.Fprintf(os.Stderr, "使用 'devflow-skills list' 查看所有可用技能\n")
				os.Exit(1)
			}
			if len(matched) > 1 {
				fmt.Fprintf(os.Stderr, "错误: 技能名 '%s' 匹配到多个技能:\n", name)
				for _, s := range matched {
					fmt.Fprintf(os.Stderr, "  %s-%s: %s\n", s.Plugin, s.Name, s.Desc)
				}
				fmt.Fprintf(os.Stderr, "请使用完整的扁平名 (如 req-dev) 来区分\n")
				os.Exit(1)
			}
			selected = append(selected, matched[0])
		}
	}

	installed := 0
	for _, s := range selected {
		content, err := skillsFS.ReadFile(s.RawPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取技能 %s-%s 失败: %v\n", s.Plugin, s.Name, err)
			continue
		}
		if err := installSkill(*tool, *dir, s.Plugin, s.Name, content); err != nil {
			fmt.Fprintf(os.Stderr, "安装 %s-%s 失败: %v\n", s.Plugin, s.Name, err)
			continue
		}
		installed++
		if !*allFlag || installed <= 10 {
			fmt.Printf("  %s-%s (%s)\n", s.Plugin, s.Name, s.Desc)
		} else if installed == 11 {
			fmt.Println("  ...")
		}
	}

	fmt.Printf("\n已安装 %d 个技能到 %s/\n", installed, targetBase(*tool, *dir))
	fmt.Println("下一步: 重启 AI 工具或刷新技能列表即可使用。")
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	plugin := fs.String("plugin", "", "按插件过滤: req / api / pm / diag / uat")
	format := fs.String("format", "text", "输出格式: text / json")

	fs.Parse(args)

	allSkills, err := loadAllSkills()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载技能失败: %v\n", err)
		os.Exit(1)
	}

	type pluginGroup struct {
		Plugin string
		Skills []skillInfo
	}

	groups := make(map[string]*pluginGroup)
	for _, s := range allSkills {
		if *plugin != "" && s.Plugin != *plugin {
			continue
		}
		if g, ok := groups[s.Plugin]; ok {
			g.Skills = append(g.Skills, s)
		} else {
			groups[s.Plugin] = &pluginGroup{Plugin: s.Plugin, Skills: []skillInfo{s}}
		}
	}

	pluginOrder := []string{"req", "api", "pm", "diag", "uat"}
	if *plugin != "" {
		pluginOrder = []string{*plugin}
	}

	if *format == "json" {
		output := make(map[string][]map[string]string)
		for _, p := range pluginOrder {
			g, ok := groups[p]
			if !ok {
				continue
			}
			skills := make([]map[string]string, 0, len(g.Skills))
			for _, s := range g.Skills {
				skills = append(skills, map[string]string{
					"name":        s.Name,
					"flatName":    s.Plugin + "-" + s.Name,
					"description": s.Desc,
				})
			}
			output[p] = skills
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
		return
	}

	for _, p := range pluginOrder {
		g, ok := groups[p]
		if !ok {
			continue
		}
		fmt.Printf("%s 插件 (%d 个技能):\n", pluginLabel(p), len(g.Skills))
		for _, s := range g.Skills {
			fmt.Printf("  %-22s %s\n", s.Plugin+"-"+s.Name, s.Desc)
		}
		fmt.Println()
	}
}

func runUninstall(args []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	tool := fs.String("tool", "", "目标 AI 工具: opencode / claude / codex")
	allFlag := fs.Bool("all", false, "卸载所有技能")
	dir := fs.String("dir", ".", "目标项目根目录")
	var skills skillsFlag
	fs.Var(&skills, "skill", "要卸载的技能名")

	fs.Parse(args)

	if *tool == "" {
		fmt.Fprintln(os.Stderr, "错误: --tool 是必需参数")
		os.Exit(1)
	}
	if !tools[*tool] {
		fmt.Fprintf(os.Stderr, "错误: 不支持的 --tool 值: %s (支持: opencode, claude, codex)\n", *tool)
		os.Exit(1)
	}
	if !*allFlag && len(skills) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 请指定 --skill 或 --all")
		os.Exit(1)
	}
	if *allFlag && len(skills) > 0 {
		fmt.Fprintln(os.Stderr, "错误: --skill 与 --all 不能同时使用")
		os.Exit(1)
	}

	if *allFlag {
		base := targetBase(*tool, *dir)
		absBase, err := filepath.Abs(base)
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取路径失败: %v\n", err)
			os.Exit(1)
		}
		if _, err := os.Stat(absBase); os.IsNotExist(err) {
			fmt.Printf("技能目录不存在: %s\n", absBase)
			fmt.Println("无需卸载。")
			return
		}
		if err := os.RemoveAll(absBase); err != nil {
			fmt.Fprintf(os.Stderr, "删除目录失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已卸载所有技能，删除目录: %s\n", absBase)
		return
	}

	removed := 0
	for _, name := range skills {
		parts := strings.SplitN(name, "-", 2)
		var plugin, skillName string
		if len(parts) == 2 {
			plugin = parts[0]
			skillName = parts[1]
		} else {
			allSkills, err := loadAllSkills()
			if err != nil {
				fmt.Fprintf(os.Stderr, "加载技能失败: %v\n", err)
				os.Exit(1)
			}
			matched := resolveSkill(name, allSkills)
			if len(matched) != 1 {
				fmt.Fprintf(os.Stderr, "错误: 无法解析技能名 '%s'，请使用扁平名如 req-dev\n", name)
				os.Exit(1)
			}
			plugin = matched[0].Plugin
			skillName = matched[0].Name
		}

		target := targetPath(*tool, *dir, plugin, skillName)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			fmt.Printf("未安装: %s-%s\n", plugin, skillName)
			continue
		}
		if err := os.RemoveAll(target); err != nil {
			fmt.Fprintf(os.Stderr, "卸载 %s-%s 失败: %v\n", plugin, skillName, err)
			continue
		}
		fmt.Printf("已卸载: %s-%s\n", plugin, skillName)
		removed++
	}

	if removed == 0 {
		fmt.Println("没有需要卸载的技能。")
	} else {
		fmt.Printf("\n已卸载 %d 个技能。\n", removed)
	}
}

func loadAllSkills() ([]skillInfo, error) {
	var skills []skillInfo
	err := fs.WalkDir(skillsFS, "plugins", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "SKILL.md" {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(path), "/")
		if len(parts) < 5 {
			return nil
		}
		plugin := parts[1]
		skillName := parts[3]

		f, err := skillsFS.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		desc := parseFrontmatterDescription(f)

		skills = append(skills, skillInfo{
			Plugin:  plugin,
			Name:    skillName,
			Desc:    desc,
			RawPath: path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Plugin != skills[j].Plugin {
			return skills[i].Plugin < skills[j].Plugin
		}
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func parseFrontmatterDescription(f fs.File) string {
	scanner := bufio.NewScanner(f)
	inFM := false
	fmClosed := false
	var fmLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if !inFM {
			if strings.TrimSpace(line) == "---" {
				inFM = true
			} else if strings.TrimSpace(line) != "" {
				fmClosed = true
				break
			}
			continue
		}
		if strings.TrimSpace(line) == "---" {
			fmClosed = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !fmClosed {
		return ""
	}

	content := strings.Join(fmLines, "\n")
	lines := strings.Split(content, "\n")

	inDesc := false
	descIndent := ""
	var descLines []string

	for _, line := range lines {
		if inDesc {
			if strings.HasPrefix(line, descIndent+"  ") || strings.HasPrefix(line, descIndent+"\t") {
				descLines = append(descLines, strings.TrimSpace(line))
			} else if strings.TrimSpace(line) == "" && len(descLines) > 0 {
				descLines = append(descLines, "")
			} else if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "description:") {
			inDesc = true
			rest := strings.TrimPrefix(line, "description:")
			if strings.HasPrefix(rest, " |") {
				descIndent = ""
			} else {
				desc := strings.TrimSpace(rest)
				if desc != "" {
					return desc
				}
			}
		}
	}

	desc := strings.TrimSpace(strings.Join(descLines, " "))
	desc = strings.Replace(desc, "\n", " ", -1)
	return desc
}

func resolveSkill(name string, allSkills []skillInfo) []skillInfo {
	parts := strings.SplitN(name, "-", 2)
	if len(parts) == 2 {
		for _, s := range allSkills {
			if s.Plugin == parts[0] && s.Name == parts[1] {
				return []skillInfo{s}
			}
		}
		return nil
	}

	var matched []skillInfo
	for _, s := range allSkills {
		if s.Name == name {
			matched = append(matched, s)
		}
	}
	return matched
}

func installSkill(tool, dir, plugin, name string, content []byte) error {
	switch tool {
	case "opencode":
		return installOpecode(dir, plugin, name, content)
	case "codex":
		return installCodex(dir, plugin, name, content)
	case "claude":
		return installClaude(dir, plugin, name, content)
	default:
		return fmt.Errorf("不支持的工具: %s", tool)
	}
}

func targetBase(tool, dir string) string {
	switch tool {
	case "opencode", "codex":
		return filepath.Join(dir, ".agents", "skills")
	case "claude":
		return filepath.Join(dir, "plugins")
	}
	return dir
}

func targetPath(tool, dir, plugin, name string) string {
	switch tool {
	case "opencode", "codex":
		return filepath.Join(dir, ".agents", "skills", plugin+"-"+name)
	case "claude":
		return filepath.Join(dir, "plugins", plugin, "skills", name)
	}
	return filepath.Join(dir, plugin, name)
}

func installOpecode(dir, plugin, name string, content []byte) error {
	flatName := plugin + "-" + name
	target := filepath.Join(dir, ".agents", "skills", flatName)
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	updated := updateFrontmatterName(content, flatName)
	return os.WriteFile(filepath.Join(target, "SKILL.md"), updated, 0644)
}

func installCodex(dir, plugin, name string, content []byte) error {
	return installOpecode(dir, plugin, name, content)
}

func installClaude(dir, plugin, name string, content []byte) error {
	target := filepath.Join(dir, "plugins", plugin, "skills", name)
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(target, "SKILL.md"), content, 0644)
}

func updateFrontmatterName(content []byte, newName string) []byte {
	lines := bytes.Split(content, []byte("\n"))
	inFM := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if !inFM {
			if trimmed == "---" {
				inFM = true
			}
			continue
		}
		if trimmed == "---" {
			break
		}
		if strings.HasPrefix(trimmed, "name:") {
			indent := ""
			for _, c := range string(line) {
				if c == ' ' || c == '\t' {
					indent += string(c)
				} else {
					break
				}
			}
			lines[i] = []byte(indent + "name: " + newName)
			break
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

func pluginLabel(p string) string {
	switch p {
	case "req":
		return "req"
	case "api":
		return "api"
	case "pm":
		return "pm"
	case "diag":
		return "diag"
	case "uat":
		return "uat"
	}
	return p
}

type skillsFlag []string

func (s *skillsFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *skillsFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}
