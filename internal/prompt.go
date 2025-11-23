package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// PromptFieldChoice 提示用户选择字段值.
func PromptFieldChoice(conflict FieldConflict, currentIndex, totalCount int) (string, string, error) {
	fmt.Printf("\n冲突 %d/%d: %s\n", currentIndex, totalCount, conflict.FieldName)
	fmt.Printf("────────────────────────────────────────\n")

	localDisplay := conflict.LocalValue
	remoteDisplay := conflict.RemoteValue

	// 如果是 APIKey 字段，遮蔽显示
	if conflict.FieldName == FieldNameAPIKey {
		localDisplay = maskAPIKeyDisplay(conflict.LocalValue)
		remoteDisplay = maskAPIKeyDisplay(conflict.RemoteValue)
	}

	fmt.Printf("  本地值:  %s\n", localDisplay)
	if !conflict.LocalTime.IsZero() {
		fmt.Printf("  修改时间: %s (本设备)\n", formatTimeAgo(conflict.LocalTime))
	}
	fmt.Println()

	fmt.Printf("  远程值:  %s\n", remoteDisplay)
	if !conflict.RemoteTime.IsZero() {
		deviceInfo := ""
		if conflict.RemoteDevice != "" {
			deviceInfo = fmt.Sprintf(" (设备: %s)", conflict.RemoteDevice)
		}
		fmt.Printf("  修改时间: %s%s\n", formatTimeAgo(conflict.RemoteTime), deviceInfo)
	}
	fmt.Println()

	fmt.Printf("选择要保留的值:\n")
	fmt.Printf("  [1] 本地 (%s)\n", localDisplay)
	fmt.Printf("  [2] 远程 (%s)\n", remoteDisplay)
	if conflict.FieldName != FieldNameAPIKey && conflict.FieldName != FieldNameToolType {
		fmt.Printf("  [3] 手动输入新值\n")
	}
	fmt.Printf("  [s] 跳过此字段（保持本地）\n")
	fmt.Printf("\n您的选择: ")

	reader := bufio.NewReader(os.Stdin)
	choice, err := reader.ReadString('\n')
	if err != nil {
		return conflict.LocalValue, StrategyLocal, nil
	}
	choice = strings.TrimSpace(choice)

	switch strings.ToLower(choice) {
	case "1":
		return conflict.LocalValue, StrategyLocal, nil
	case "2":
		return conflict.RemoteValue, StrategyRemote, nil
	case "3":
		if conflict.FieldName != FieldNameAPIKey && conflict.FieldName != FieldNameToolType {
			value, err := promptManualInput(conflict.FieldName)
			if err != nil {
				return conflict.LocalValue, StrategyLocal, nil
			}
			return value, StrategyManual, nil
		}
		fmt.Printf("⚠️  此字段不支持手动输入，默认保留本地值\n")
		return conflict.LocalValue, StrategyLocal, nil
	case "s", "":
		return conflict.LocalValue, StrategyLocal, nil
	default:
		fmt.Printf("⚠️  无效选择，默认保留本地值\n")
		return conflict.LocalValue, StrategyLocal, nil
	}
}

// promptManualInput 提示用户手动输入值.
func promptManualInput(fieldName string) (string, error) {
	fmt.Printf("请输入新的 %s 值: ", fieldName)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// PromptConfirmation 提示用户确认操作.
func PromptConfirmation(message string) bool {
	fmt.Printf("%s [y/n]: ", message)
	reader := bufio.NewReader(os.Stdin)
	choice, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	choice = strings.TrimSpace(strings.ToLower(choice))
	return choice == "y" || choice == "yes"
}

// ShowMergeResult 显示合并结果.
func ShowMergeResult(mirrorName string, resolutions []FieldResolution) {
	fmt.Printf("\n✅ 镜像源 '%s' 冲突已解决，合并结果：\n", mirrorName)
	fmt.Printf("────────────────────────────────────────\n")

	for _, res := range resolutions {
		displayValue := res.ResolvedValue
		if res.FieldName == FieldNameAPIKey {
			displayValue = maskAPIKeyDisplay(res.ResolvedValue)
		}

		choiceLabel := ""
		switch res.Choice {
		case StrategyLocal:
			choiceLabel = "本地"
		case StrategyRemote:
			choiceLabel = "远程"
		case StrategyManual:
			choiceLabel = "手动"
		case StrategyAuto:
			choiceLabel = "自动"
		}

		fmt.Printf("  %-12s %s (%s)\n", res.FieldName+":", displayValue, choiceLabel)
	}
	fmt.Println()
}

// formatTimeAgo 格式化时间为"多久前"的形式.
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "未知"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "刚刚"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d分钟前", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%d小时前", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d天前", days)
	default:
		return t.Format("2006-01-02 15:04")
	}
}

// maskAPIKeyDisplay 遮蔽API密钥显示，只显示前4位和后4位.
func maskAPIKeyDisplay(apiKey string) string {
	if apiKey == "" {
		return "(空)"
	}
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// PrintConflictHeader 打印冲突解决的标题.
func PrintConflictHeader(mirrorName string, conflictCount int) {
	fmt.Printf("\n⚠️  检测到并发修改冲突！\n\n")
	fmt.Printf("镜像源: %s (共 %d 个字段冲突)\n", mirrorName, conflictCount)
	fmt.Printf("════════════════════════════════════════\n")
}

// PrintAutoMergeInfo 打印自动合并信息.
func PrintAutoMergeInfo(fieldName, value, reason string) {
	displayValue := value
	if fieldName == FieldNameAPIKey {
		displayValue = maskAPIKeyDisplay(value)
	}
	fmt.Printf("🔄 自动合并 %s: %s (%s)\n", fieldName, displayValue, reason)
}
