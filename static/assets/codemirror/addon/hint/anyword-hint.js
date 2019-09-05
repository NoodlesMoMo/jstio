(function(mod) {
    if (typeof exports == "object" && typeof module == "object") // CommonJS
        mod(require("../../lib/codemirror"));
    else if (typeof define == "function" && define.amd) // AMD
        define(["../../lib/codemirror"], mod);
    else
        mod(CodeMirror);
})(function(CodeMirror) {
    var Pos = CodeMirror.Pos; // A Pos instance represents a position within the text.

    function arrayContains(arr, item) { // 判断元素item是否存在数组arr中
        if (!Array.prototype.indexOf) {
            var i = arr.length;
            while (i--) {
                if (arr[i] === item) {
                    return true;
                }
            }
            return false;
        }
        return arr.indexOf(item) != -1;
    }

    function scriptHint(editor, context, keywords, getToken, options) { // 处理hint的核心函数，改名为velocityHint(也可以不做修改)
        // Find the token at the cursor，获取当前光标指定的字符串
        var cur = editor.getCursor(), token = getToken(editor, cur), tprop = token;
        return {
            list : getCompletions(token, context, keywords, options),
            from : Pos(cur.line, fetchStartPoint(token)), // 字符串拼接的初始位置，这个很重要
            to : Pos(cur.line, token.end)
        };
    }

    function fetchStartPoint(token) {
        var index = token.string.lastIndexOf("\.");
        if (index < 0) {
            return token.start + 1;
        } else {
            return token.start + index + 1;
        }
    }

    function velocityHint(editor, options) {
        return scriptHint(editor, CodeMirror.velocityContext, CodeMirror.velocityCustomizedKeywords, function(e, cur) {
            return e.getTokenAt(cur);
        }, options);
    }

    CodeMirror.registerHelper("hint", "yaml", velocityHint);
    function getCompletions(token, context, keywords, options) {
        var found = [], start, pointCount, content = token.string; // found为匹配的数组
        if (content && content.length) {
            start = token.string.charAt(0);
            content = content.substring(1, content.lenght);
            pointCount = content.split('.').length - 1; // 查看字符串中有多少个.
        }
        var result = null;
        if (start == '$') { // 必须以$开头，这里暂时不解析${}
            var regexp = new RegExp("\\b" + content + "\\w+\\.?\\b", "gi");
            if (pointCount == 0) {
                result = context.match(regexp);
            } else {
                result = keywords.match(regexp);
            }
        }
        if (result && result.length) {
            for (var i = 0; i < result.length; i++) {
                if (!arrayContains(found, result[i]) && pointCount > 0) {
                    found.push(result[i].substring(content.lastIndexOf("\.") + 1, result[i].length));
                } else {
                    found.push(result[i]);
                }
            }
        }
        return found;
    }
});
