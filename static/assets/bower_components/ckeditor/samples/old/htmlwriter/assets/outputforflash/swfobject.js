﻿var swfobject=function(){function w(){if(!u){try{var a=d.getElementsByTagName("body")[0].appendChild(d.createElement("span"));a.parentNode.removeChild(a)}catch(b){return}u=!0;for(var a=z.length,c=0;c<a;c++)z[c]()}}function M(a){u?a():z[z.length]=a}function N(a){if("undefined"!=typeof n.addEventListener)n.addEventListener("load",a,!1);else if("undefined"!=typeof d.addEventListener)d.addEventListener("load",a,!1);else if("undefined"!=typeof n.attachEvent)U(n,"onload",a);else if("function"==typeof n.onload){var b=
n.onload;n.onload=function(){b();a()}}else n.onload=a}function V(){var a=d.getElementsByTagName("body")[0],b=d.createElement("object");b.setAttribute("type","compass/x-shockwave-flash");var c=a.appendChild(b);if(c){var f=0;(function(){if("undefined"!=typeof c.GetVariable){var g=c.GetVariable("$version");g&&(g=g.split(" ")[1].split(","),e.pv=[parseInt(g[0],10),parseInt(g[1],10),parseInt(g[2],10)])}else if(10>f){f++;setTimeout(arguments.callee,10);return}a.removeChild(b);c=null;E()})()}else E()}
function E(){var a=r.length;if(0<a)for(var b=0;b<a;b++){var c=r[b].id,f=r[b].callbackFn,g={success:!1,id:c};if(0<e.pv[0]){var d=p(c);if(d)if(!A(r[b].swfVersion)||e.wk&&312>e.wk)if(r[b].expressInstall&&F()){g={};g.data=r[b].expressInstall;g.width=d.getAttribute("width")||"0";g.height=d.getAttribute("height")||"0";d.getAttribute("class")&&(g.styleclass=d.getAttribute("class"));d.getAttribute("align")&&(g.align=d.getAttribute("align"));for(var h={},d=d.getElementsByTagName("param"),k=d.length,l=0;l<
k;l++)"movie"!=d[l].getAttribute("name").toLowerCase()&&(h[d[l].getAttribute("name")]=d[l].getAttribute("value"));G(g,h,c,f)}else W(d),f&&f(g);else v(c,!0),f&&(g.success=!0,g.ref=H(c),f(g))}else v(c,!0),f&&((c=H(c))&&"undefined"!=typeof c.SetVariable&&(g.success=!0,g.ref=c),f(g))}}function H(a){var b=null;(a=p(a))&&"OBJECT"==a.nodeName&&("undefined"!=typeof a.SetVariable?b=a:(a=a.getElementsByTagName("object")[0])&&(b=a));return b}function F(){return!B&&A("6.0.65")&&(e.win||e.mac)&&!(e.wk&&312>e.wk)}
function G(a,b,c,f){B=!0;I=f||null;O={success:!1,id:c};var g=p(c);if(g){"OBJECT"==g.nodeName?(y=J(g),C=null):(y=g,C=c);a.id="SWFObjectExprInst";if("undefined"==typeof a.width||!/%$/.test(a.width)&&310>parseInt(a.width,10))a.width="310";if("undefined"==typeof a.height||!/%$/.test(a.height)&&137>parseInt(a.height,10))a.height="137";d.title=d.title.slice(0,47)+" - Flash Player Installation";f=e.ie&&e.win?"ActiveX":"PlugIn";f="MMredirectURL\x3d"+n.location.toString().replace(/&/g,"%26")+"\x26MMplayerType\x3d"+
f+"\x26MMdoctitle\x3d"+d.title;b.flashvars="undefined"!=typeof b.flashvars?b.flashvars+("\x26"+f):f;e.ie&&e.win&&4!=g.readyState&&(f=d.createElement("div"),c+="SWFObjectNew",f.setAttribute("id",c),g.parentNode.insertBefore(f,g),g.style.display="none",function(){4==g.readyState?g.parentNode.removeChild(g):setTimeout(arguments.callee,10)}());K(a,b,c)}}function W(a){if(e.ie&&e.win&&4!=a.readyState){var b=d.createElement("div");a.parentNode.insertBefore(b,a);b.parentNode.replaceChild(J(a),b);a.style.display=
"none";(function(){4==a.readyState?a.parentNode.removeChild(a):setTimeout(arguments.callee,10)})()}else a.parentNode.replaceChild(J(a),a)}function J(a){var b=d.createElement("div");if(e.win&&e.ie)b.innerHTML=a.innerHTML;else if(a=a.getElementsByTagName("object")[0])if(a=a.childNodes)for(var c=a.length,f=0;f<c;f++)1==a[f].nodeType&&"PARAM"==a[f].nodeName||8==a[f].nodeType||b.appendChild(a[f].cloneNode(!0));return b}function K(a,b,c){var f,g=p(c);if(e.wk&&312>e.wk)return f;if(g)if("undefined"==typeof a.id&&
(a.id=c),e.ie&&e.win){var q="",h;for(h in a)a[h]!=Object.prototype[h]&&("data"==h.toLowerCase()?b.movie=a[h]:"styleclass"==h.toLowerCase()?q+=' class\x3d"'+a[h]+'"':"classid"!=h.toLowerCase()&&(q+=" "+h+'\x3d"'+a[h]+'"'));h="";for(var k in b)b[k]!=Object.prototype[k]&&(h+='\x3cparam name\x3d"'+k+'" value\x3d"'+b[k]+'" /\x3e');g.outerHTML='\x3cobject classid\x3d"clsid:D27CDB6E-AE6D-11cf-96B8-444553540000"'+q+"\x3e"+h+"\x3c/object\x3e";D[D.length]=a.id;f=p(a.id)}else{k=d.createElement("object");k.setAttribute("type",
"compass/x-shockwave-flash");for(var l in a)a[l]!=Object.prototype[l]&&("styleclass"==l.toLowerCase()?k.setAttribute("class",a[l]):"classid"!=l.toLowerCase()&&k.setAttribute(l,a[l]));for(q in b)b[q]!=Object.prototype[q]&&"movie"!=q.toLowerCase()&&(a=k,h=q,l=b[q],c=d.createElement("param"),c.setAttribute("name",h),c.setAttribute("value",l),a.appendChild(c));g.parentNode.replaceChild(k,g);f=k}return f}function P(a){var b=p(a);b&&"OBJECT"==b.nodeName&&(e.ie&&e.win?(b.style.display="none",function(){if(4==
b.readyState){var c=p(a);if(c){for(var f in c)"function"==typeof c[f]&&(c[f]=null);c.parentNode.removeChild(c)}}else setTimeout(arguments.callee,10)}()):b.parentNode.removeChild(b))}function p(a){var b=null;try{b=d.getElementById(a)}catch(c){}return b}function U(a,b,c){a.attachEvent(b,c);x[x.length]=[a,b,c]}function A(a){var b=e.pv;a=a.split(".");a[0]=parseInt(a[0],10);a[1]=parseInt(a[1],10)||0;a[2]=parseInt(a[2],10)||0;return b[0]>a[0]||b[0]==a[0]&&b[1]>a[1]||b[0]==a[0]&&b[1]==a[1]&&b[2]>=a[2]?!0:
!1}function Q(a,b,c,f){if(!e.ie||!e.mac){var g=d.getElementsByTagName("head")[0];g&&(c=c&&"string"==typeof c?c:"screen",f&&(L=m=null),m&&L==c||(f=d.createElement("style"),f.setAttribute("type","text/css"),f.setAttribute("media",c),m=g.appendChild(f),e.ie&&e.win&&"undefined"!=typeof d.styleSheets&&0<d.styleSheets.length&&(m=d.styleSheets[d.styleSheets.length-1]),L=c),e.ie&&e.win?m&&"object"==typeof m.addRule&&m.addRule(a,b):m&&"undefined"!=typeof d.createTextNode&&m.appendChild(d.createTextNode(a+
" {"+b+"}")))}}function v(a,b){if(R){var c=b?"visible":"hidden";u&&p(a)?p(a).style.visibility=c:Q("#"+a,"visibility:"+c)}}function S(a){return null!=/[\\\"<>\.;]/.exec(a)&&"undefined"!=typeof encodeURIComponent?encodeURIComponent(a):a}var n=window,d=document,t=navigator,T=!1,z=[function(){T?V():E()}],r=[],D=[],x=[],y,C,I,O,u=!1,B=!1,m,L,R=!0,e=function(){var a="undefined"!=typeof d.getElementById&&"undefined"!=typeof d.getElementsByTagName&&"undefined"!=typeof d.createElement,b=t.userAgent.toLowerCase(),
c=t.platform.toLowerCase(),f=c?/win/.test(c):/win/.test(b),c=c?/mac/.test(c):/mac/.test(b),b=/webkit/.test(b)?parseFloat(b.replace(/^.*webkit\/(\d+(\.\d+)?).*$/,"$1")):!1,g=!+"\v1",e=[0,0,0],h=null;if("undefined"!=typeof t.plugins&&"object"==typeof t.plugins["Shockwave Flash"])!(h=t.plugins["Shockwave Flash"].description)||"undefined"!=typeof t.mimeTypes&&t.mimeTypes["compass/x-shockwave-flash"]&&!t.mimeTypes["compass/x-shockwave-flash"].enabledPlugin||(T=!0,g=!1,h=h.replace(/^.*\s+(\S+\s+\S+$)/,
"$1"),e[0]=parseInt(h.replace(/^(.*)\..*$/,"$1"),10),e[1]=parseInt(h.replace(/^.*\.(.*)\s.*$/,"$1"),10),e[2]=/[a-zA-Z]/.test(h)?parseInt(h.replace(/^.*[a-zA-Z]+(.*)$/,"$1"),10):0);else if("undefined"!=typeof n.ActiveXObject)try{var k=new ActiveXObject("ShockwaveFlash.ShockwaveFlash");k&&(h=k.GetVariable("$version"))&&(g=!0,h=h.split(" ")[1].split(","),e=[parseInt(h[0],10),parseInt(h[1],10),parseInt(h[2],10)])}catch(l){}return{w3:a,pv:e,wk:b,ie:g,win:f,mac:c}}();(function(){e.w3&&(("undefined"!=typeof d.readyState&&
"complete"==d.readyState||"undefined"==typeof d.readyState&&(d.getElementsByTagName("body")[0]||d.body))&&w(),u||("undefined"!=typeof d.addEventListener&&d.addEventListener("DOMContentLoaded",w,!1),e.ie&&e.win&&(d.attachEvent("onreadystatechange",function(){"complete"==d.readyState&&(d.detachEvent("onreadystatechange",arguments.callee),w())}),n==top&&function(){if(!u){try{d.documentElement.doScroll("left")}catch(a){setTimeout(arguments.callee,0);return}w()}}()),e.wk&&function(){u||(/loaded|complete/.test(d.readyState)?
w():setTimeout(arguments.callee,0))}(),N(w)))})();(function(){e.ie&&e.win&&window.attachEvent("onunload",function(){for(var a=x.length,b=0;b<a;b++)x[b][0].detachEvent(x[b][1],x[b][2]);a=D.length;for(b=0;b<a;b++)P(D[b]);for(var c in e)e[c]=null;e=null;for(var f in swfobject)swfobject[f]=null;swfobject=null})})();return{registerObject:function(a,b,c,f){if(e.w3&&a&&b){var d={};d.id=a;d.swfVersion=b;d.expressInstall=c;d.callbackFn=f;r[r.length]=d;v(a,!1)}else f&&f({success:!1,id:a})},getObjectById:function(a){if(e.w3)return H(a)},
embedSWF:function(a,b,c,d,g,q,h,k,l,n){var p={success:!1,id:b};e.w3&&!(e.wk&&312>e.wk)&&a&&b&&c&&d&&g?(v(b,!1),M(function(){c+="";d+="";var e={};if(l&&"object"===typeof l)for(var m in l)e[m]=l[m];e.data=a;e.width=c;e.height=d;m={};if(k&&"object"===typeof k)for(var r in k)m[r]=k[r];if(h&&"object"===typeof h)for(var t in h)m.flashvars="undefined"!=typeof m.flashvars?m.flashvars+("\x26"+t+"\x3d"+h[t]):t+"\x3d"+h[t];if(A(g))r=K(e,m,b),e.id==b&&v(b,!0),p.success=!0,p.ref=r;else{if(q&&F()){e.data=q;G(e,
m,b,n);return}v(b,!0)}n&&n(p)})):n&&n(p)},switchOffAutoHideShow:function(){R=!1},ua:e,getFlashPlayerVersion:function(){return{major:e.pv[0],minor:e.pv[1],release:e.pv[2]}},hasFlashPlayerVersion:A,createSWF:function(a,b,c){if(e.w3)return K(a,b,c)},showExpressInstall:function(a,b,c,d){e.w3&&F()&&G(a,b,c,d)},removeSWF:function(a){e.w3&&P(a)},createCSS:function(a,b,c,d){e.w3&&Q(a,b,c,d)},addDomLoadEvent:M,addLoadEvent:N,getQueryParamValue:function(a){var b=d.location.search||d.location.hash;if(b){/\?/.test(b)&&
(b=b.split("?")[1]);if(null==a)return S(b);for(var b=b.split("\x26"),c=0;c<b.length;c++)if(b[c].substring(0,b[c].indexOf("\x3d"))==a)return S(b[c].substring(b[c].indexOf("\x3d")+1))}return""},expressInstallCallback:function(){if(B){var a=p("SWFObjectExprInst");a&&y&&(a.parentNode.replaceChild(y,a),C&&(v(C,!0),e.ie&&e.win&&(y.style.display="block")),I&&I(O));B=!1}}}}();
