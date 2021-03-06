package extend

import (
	"bytes"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	c "github.com/Azareal/Gosora/common"
)

var bbcodeRandom *rand.Rand
var bbcodeInvalidNumber []byte
var bbcodeNoNegative []byte
var bbcodeMissingTag []byte

var bbcodeBold *regexp.Regexp
var bbcodeItalic *regexp.Regexp
var bbcodeUnderline *regexp.Regexp
var bbcodeStrike *regexp.Regexp
var bbcodeH1 *regexp.Regexp
var bbcodeURL *regexp.Regexp
var bbcodeURLLabel *regexp.Regexp
var bbcodeQuotes *regexp.Regexp
var bbcodeCode *regexp.Regexp
var bbcodeSpoiler *regexp.Regexp

func init() {
	c.Plugins.Add(&c.Plugin{UName: "bbcode", Name: "BBCode", Author: "Azareal", URL: "https://github.com/Azareal", Init: InitBbcode, Deactivate: deactivateBbcode})
}

func InitBbcode(pl *c.Plugin) error {
	pl.AddHook("parse_assign", BbcodeFullParse)

	bbcodeInvalidNumber = []byte("<red>[Invalid Number]</red>")
	bbcodeNoNegative = []byte("<red>[No Negative Numbers]</red>")
	bbcodeMissingTag = []byte("<red>[Missing Tag]</red>")

	bbcodeBold = regexp.MustCompile(`(?s)\[b\](.*)\[/b\]`)
	bbcodeItalic = regexp.MustCompile(`(?s)\[i\](.*)\[/i\]`)
	bbcodeUnderline = regexp.MustCompile(`(?s)\[u\](.*)\[/u\]`)
	bbcodeStrike = regexp.MustCompile(`(?s)\[s\](.*)\[/s\]`)
	bbcodeH1 = regexp.MustCompile(`(?s)\[h1\](.*)\[/h1\]`)
	urlpattern := `(http|https|ftp|mailto*)(:??)\/\/([\.a-zA-Z\/]+)`
	bbcodeURL = regexp.MustCompile(`\[url\]` + urlpattern + `\[/url\]`)
	bbcodeURLLabel = regexp.MustCompile(`(?s)\[url=` + urlpattern + `\](.*)\[/url\]`)
	bbcodeQuotes = regexp.MustCompile(`\[quote\](.*)\[/quote\]`)
	bbcodeCode = regexp.MustCompile(`\[code\](.*)\[/code\]`)
	bbcodeSpoiler = regexp.MustCompile(`\[spoiler\](.*)\[/spoiler\]`)

	bbcodeRandom = rand.New(rand.NewSource(time.Now().UnixNano()))
	return nil
}

func deactivateBbcode(pl *c.Plugin) {
	pl.RemoveHook("parse_assign", BbcodeFullParse)
}

func BbcodeRegexParse(msg string) string {
	msg = bbcodeBold.ReplaceAllString(msg, "<b>$1</b>")
	msg = bbcodeItalic.ReplaceAllString(msg, "<i>$1</i>")
	msg = bbcodeUnderline.ReplaceAllString(msg, "<u>$1</u>")
	msg = bbcodeStrike.ReplaceAllString(msg, "<s>$1</s>")
	msg = bbcodeURL.ReplaceAllString(msg, "<a href=''$1$2//$3' rel='ugc'>$1$2//$3</i>")
	msg = bbcodeURLLabel.ReplaceAllString(msg, "<a href=''$1$2//$3' rel='ugc'>$4</i>")
	msg = bbcodeQuotes.ReplaceAllString(msg, "<blockquote>$1</blockquote>")
	msg = bbcodeSpoiler.ReplaceAllString(msg, "<spoiler>$1</spoiler>")
	msg = bbcodeH1.ReplaceAllString(msg, "<h2>$1</h2>")
	//msg = bbcodeCode.ReplaceAllString(msg,"<span class='codequotes'>$1</span>")
	return msg
}

// Only does the simple BBCode like [u], [b], [i] and [s]
func bbcodeSimpleParse(msg string) string {
	var hasU, hasB, hasI, hasS bool
	mbytes := []byte(msg)
	for i := 0; (i + 2) < len(mbytes); i++ {
		if mbytes[i] == '[' && mbytes[i+2] == ']' {
			ch := mbytes[i+1]
			if ch == 'b' && !hasB {
				mbytes[i] = '<'
				mbytes[i+2] = '>'
				hasB = true
			} else if ch == 'i' && !hasI {
				mbytes[i] = '<'
				mbytes[i+2] = '>'
				hasI = true
			} else if ch == 'u' && !hasU {
				mbytes[i] = '<'
				mbytes[i+2] = '>'
				hasU = true
			} else if ch == 's' && !hasS {
				mbytes[i] = '<'
				mbytes[i+2] = '>'
				hasS = true
			}
			i += 2
		}
	}

	// There's an unclosed tag in there x.x
	if hasI || hasU || hasB || hasS {
		closeUnder := []byte("</u>")
		closeItalic := []byte("</i>")
		closeBold := []byte("</b>")
		closeStrike := []byte("</s>")
		if hasI {
			mbytes = append(mbytes, closeItalic...)
		}
		if hasU {
			mbytes = append(mbytes, closeUnder...)
		}
		if hasB {
			mbytes = append(mbytes, closeBold...)
		}
		if hasS {
			mbytes = append(mbytes, closeStrike...)
		}
	}
	return string(mbytes)
}

// Here for benchmarking purposes. Might add a plugin setting for disabling [code] as it has it's paws everywhere
func BbcodeParseWithoutCode(msg string) string {
	var hasU, hasB, hasI, hasS bool
	var complexBbc bool
	mbytes := []byte(msg)
	for i := 0; (i + 3) < len(mbytes); i++ {
		if mbytes[i] == '[' {
			if mbytes[i+2] != ']' {
				if mbytes[i+1] == '/' {
					if mbytes[i+3] == ']' {
						switch mbytes[i+2] {
						case 'b':
							mbytes[i] = '<'
							mbytes[i+3] = '>'
							hasB = false
						case 'i':
							mbytes[i] = '<'
							mbytes[i+3] = '>'
							hasI = false
						case 'u':
							mbytes[i] = '<'
							mbytes[i+3] = '>'
							hasU = false
						case 's':
							mbytes[i] = '<'
							mbytes[i+3] = '>'
							hasS = false
						}
						i += 3
					} else {
						complexBbc = true
					}
				} else {
					complexBbc = true
				}
			} else {
				ch := mbytes[i+1]
				if ch == 'b' && !hasB {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasB = true
				} else if ch == 'i' && !hasI {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasI = true
				} else if ch == 'u' && !hasU {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasU = true
				} else if ch == 's' && !hasS {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasS = true
				}
				i += 2
			}
		}
	}

	// There's an unclosed tag in there x.x
	if hasI || hasU || hasB || hasS {
		closeUnder := []byte("</u>")
		closeItalic := []byte("</i>")
		closeBold := []byte("</b>")
		closeStrike := []byte("</s>")
		if hasI {
			mbytes = append(bytes.TrimSpace(mbytes), closeItalic...)
		}
		if hasU {
			mbytes = append(bytes.TrimSpace(mbytes), closeUnder...)
		}
		if hasB {
			mbytes = append(bytes.TrimSpace(mbytes), closeBold...)
		}
		if hasS {
			mbytes = append(bytes.TrimSpace(mbytes), closeStrike...)
		}
	}

	// Copy the new complex parser over once the rough edges have been smoothed over
	if complexBbc {
		msg = string(mbytes)
		msg = bbcodeURL.ReplaceAllString(msg, "<a href='$1$2//$3' rel='ugc'>$1$2//$3</i>")
		msg = bbcodeURLLabel.ReplaceAllString(msg, "<a href='$1$2//$3' rel='ugc'>$4</i>")
		msg = bbcodeSpoiler.ReplaceAllString(msg, "<spoiler>$1</spoiler>")
		msg = bbcodeQuotes.ReplaceAllString(msg, "<blockquote>$1</blockquote>")
		return bbcodeCode.ReplaceAllString(msg, "<span class='codequotes'>$1</span>")
	}
	return string(mbytes)
}

// Does every type of BBCode
func BbcodeFullParse(msg string) string {
	var hasU, hasB, hasI, hasS, hasC bool
	var complexBbc bool

	mbytes := []byte(msg)
	mbytes = append(mbytes, c.SpaceGap...)
	for i := 0; i < len(mbytes); i++ {
		if mbytes[i] == '[' {
			if mbytes[i+2] != ']' {
				if mbytes[i+1] == '/' {
					if mbytes[i+3] == ']' {
						if !hasC {
							switch mbytes[i+2] {
							case 'b':
								mbytes[i] = '<'
								mbytes[i+3] = '>'
								hasB = false
							case 'i':
								mbytes[i] = '<'
								mbytes[i+3] = '>'
								hasI = false
							case 'u':
								mbytes[i] = '<'
								mbytes[i+3] = '>'
								hasU = false
							case 's':
								mbytes[i] = '<'
								mbytes[i+3] = '>'
								hasS = false
							}
							i += 3
						}
					} else {
						if mbytes[i+6] == ']' && mbytes[i+2] == 'c' && mbytes[i+3] == 'o' && mbytes[i+4] == 'd' && mbytes[i+5] == 'e' {
							hasC = false
							i += 7
						}
						complexBbc = true
					}
				} else {
					// Put the biggest index first to avoid unnecessary bounds checks
					if mbytes[i+5] == ']' && mbytes[i+1] == 'c' && mbytes[i+2] == 'o' && mbytes[i+3] == 'd' && mbytes[i+4] == 'e' {
						hasC = true
						i += 6
					}
					complexBbc = true
				}
			} else if !hasC {
				ch := mbytes[i+1]
				if ch == 'b' && !hasB {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasB = true
				} else if ch == 'i' && !hasI {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasI = true
				} else if ch == 'u' && !hasU {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasU = true
				} else if ch == 's' && !hasS {
					mbytes[i] = '<'
					mbytes[i+2] = '>'
					hasS = true
				}
				i += 2
			}
		}
	}

	// There's an unclosed tag in there somewhere x.x
	if hasI || hasU || hasB || hasS {
		closeUnder := []byte("</u>")
		closeItalic := []byte("</i>")
		closeBold := []byte("</b>")
		closeStrike := []byte("</s>")
		if hasI {
			mbytes = append(bytes.TrimSpace(mbytes), closeItalic...)
		}
		if hasU {
			mbytes = append(bytes.TrimSpace(mbytes), closeUnder...)
		}
		if hasB {
			mbytes = append(bytes.TrimSpace(mbytes), closeBold...)
		}
		if hasS {
			mbytes = append(bytes.TrimSpace(mbytes), closeStrike...)
		}
		mbytes = append(mbytes, c.SpaceGap...)
	}

	if complexBbc {
		i := 0
		var start, lastTag int
		var outbytes []byte
		for ; i < len(mbytes); i++ {
			if mbytes[i] == '[' {
				if mbytes[i+1] == 'u' {
					if mbytes[i+4] == ']' && mbytes[i+2] == 'r' && mbytes[i+3] == 'l' {
						i, start, lastTag, outbytes = bbcodeParseURL(i, start, lastTag, mbytes, outbytes)
						continue
					}
				} else if mbytes[i+1] == 'r' {
					if bytes.Equal(mbytes[i+2:i+6], []byte("and]")) {
						i, start, lastTag, outbytes = bbcodeParseRand(i, start, lastTag, mbytes, outbytes)
					}
				}
			}
		}
		if lastTag != i {
			outbytes = append(outbytes, mbytes[lastTag:]...)
		}

		if len(outbytes) != 0 {
			msg = string(outbytes[0 : len(outbytes)-10])
		} else {
			msg = string(mbytes[0 : len(mbytes)-10])
		}

		// TODO: Optimise these
		//msg = bbcode_url.ReplaceAllString(msg,"<a href=\"$1$2//$3\" rel=\"ugc\">$1$2//$3</i>")
		msg = bbcodeURLLabel.ReplaceAllString(msg, "<a href='$1$2//$3' rel='ugc'>$4</i>")
		msg = bbcodeQuotes.ReplaceAllString(msg, "<blockquote>$1</blockquote>")
		msg = bbcodeCode.ReplaceAllString(msg, "<span class='codequotes'>$1</span>")
		msg = bbcodeSpoiler.ReplaceAllString(msg, "<spoiler>$1</spoiler>")
		msg = bbcodeH1.ReplaceAllString(msg, "<h2>$1</h2>")
	} else {
		msg = string(mbytes[0 : len(mbytes)-10])
	}

	return msg
}

// TODO: Strip the containing [url] so the media parser can work it's magic instead? Or do we want to allow something like [url=]label[/url] here?
func bbcodeParseURL(i int, start int, lastTag int, mbytes []byte, outbytes []byte) (int, int, int, []byte) {
	start = i + 5
	outbytes = append(outbytes, mbytes[lastTag:i]...)
	i = start
	i += c.PartialURLStringLen2(string(mbytes[start:]))
	if !bytes.Equal(mbytes[i:i+6], []byte("[/url]")) {
		outbytes = append(outbytes, c.InvalidURL...)
		return i, start, lastTag, outbytes
	}

	outbytes = append(outbytes, c.URLOpen...)
	outbytes = append(outbytes, mbytes[start:i]...)
	outbytes = append(outbytes, c.URLOpen2...)
	outbytes = append(outbytes, mbytes[start:i]...)
	outbytes = append(outbytes, c.URLClose...)
	i += 6
	lastTag = i

	return i, start, lastTag, outbytes
}

func bbcodeParseRand(i int, start int, lastTag int, msgbytes []byte, outbytes []byte) (int, int, int, []byte) {
	outbytes = append(outbytes, msgbytes[lastTag:i]...)
	start = i + 6
	i = start
	for ; ; i++ {
		if msgbytes[i] == '[' {
			if !bytes.Equal(msgbytes[i+1:i+7], []byte("/rand]")) {
				outbytes = append(outbytes, bbcodeMissingTag...)
				return i, start, lastTag, outbytes
			}
			break
		} else if (len(msgbytes) - 1) < (i + 10) {
			outbytes = append(outbytes, bbcodeMissingTag...)
			return i, start, lastTag, outbytes
		}
	}

	number, err := strconv.ParseInt(string(msgbytes[start:i]), 10, 64)
	if err != nil {
		outbytes = append(outbytes, bbcodeInvalidNumber...)
		return i, start, lastTag, outbytes
	}

	// TODO: Add support for negative numbers?
	if number < 0 {
		outbytes = append(outbytes, bbcodeNoNegative...)
		return i, start, lastTag, outbytes
	}

	var dat []byte
	if number == 0 {
		dat = []byte("0")
	} else {
		dat = []byte(strconv.FormatInt((bbcodeRandom.Int63n(number)), 10))
	}

	outbytes = append(outbytes, dat...)
	i += 7
	lastTag = i
	return i, start, lastTag, outbytes
}
