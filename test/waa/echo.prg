// A partir de aqui igual que con WAA
//------------------------------------------------------------------------------
function _register( oPackage )
oPackage:registerForm( "echo" )
return .T.
//------------------------------------------------------------------------------
function _version()   ; return "ECHO"
function _copyright() ; return "ECHO"
//------------------------------------------------------------------------------
function echo(html)
   local cc := '<html><head><title>echo></head><body><table>'
   aeval( html:GetAllvars() , {|kv| cc += dump_var( kv[1] , kv[2] ) } ) 
   cc += '</table></body></html>'
   html:put(cc)
   return .T.    
//------------------------------------------------------------------------------
static function dump_var(k,va)                                                  
   local cc := '<tr><td>' + k + '</td><td>'
   if len(va) == 1
   	cc += htmlenc(va[1])
   else
   	cc += '<table>'
   	aeval( va , {|v| cc += '<tr><td>' + htmlenc(v) + '</td></tr>'} )
   	cc += '</table>'
   end
   cc += '</td></tr>'	
   return cc
                                                                                
//------------------------------------------------------------------------------                                                                                
static function htmlenc(ci)        
   local co  := ''
   local nn,n,ch 
   static t := { "&euro;"   , "&#129;"   , "&sbquo;"  , "&fnof;"   , "&bdquo;"  , "&hellip;" , "&dagger;" , "&Dagger;"  ,;
                "&circ;"   , "&permil;" , "&Scaron;" , "&lsaquo;" , "&OElig;"  , "&#141;"   , "&Zcaron;" , "&#143;"    ,;
                "&#144;"   , "&lsquo;"  , "&rsquo;"  , "&ldquo;"  , "&rdquo;"  , "&bull;"   , "&ndash;"  , "&mdash;"   ,;
                "&tilde;"  , "&trade;"  , "&scaron;" , "&rsaquo;" , "&oelig;"  , "&#157;"   , "&zcaron;" , "&Yuml;"    ,;
                "&nbsp;"   , "&iexcl;"  , "&cent;"   , "&pound;"  , "&curren;" , "&yen;"    , "&brvbar;" , "&sect;"    ,;
                "&uml;"    , "&copy;"   , "&ordf;"   , "&laquo;"  , "&not;"    , "&shy;"    , "&reg;"    , "&macr;"    ,;
                "&deg;"    , "&plusmn;" , "&sup2;"   , "&sup3;"   , "&acute;"  , "&micro;"  , "&para;"   , "&middot;"  ,;
                "&cedil;"  , "&sup1;"   , "&ordm;"   , "&raquo;"  , "&frac14;" , "&frac12;" , "&frac34;" , "&iquest;"  ,;
                "&Agrave;" , "&Aacute;" , "&Acirc;"  , "&Atilde;" , "&Auml;"   , "&Aring;"  , "&AElig;"  , "&Ccedil;"  ,;
                "&Egrave;" , "&Eacute;" , "&Ecirc;"  , "&Euml;"   , "&Igrave;" , "&Iacute;" , "&Icirc;"  , "&Iuml;"    ,;
                "&ETH;"    , "&Ntilde;" , "&Ograve;" , "&Oacute;" , "&Ocirc;"  , "&Otilde;" , "&Ouml;"   , "&times;"   ,;
                "&Oslash;" , "&Ugrave;" , "&Uacute;" , "&Ucirc;"  , "&Uuml;"   , "&Yacute;" , "&THORN;"  , "&szlig;"   ,;
                "&agrave;" , "&aacute;" , "&acirc;"  , "&atilde;" , "&auml;"   , "&aring;"  , "&aelig;"  , "&ccedil;"  ,;
                "&egrave;" , "&eacute;" , "&ecirc;"  , "&euml;"   , "&igrave;" , "&iacute;" , "&icirc;"  , "&iuml;"    ,;
                "&eth;"    , "&ntilde;" , "&ograve;" , "&oacute;" , "&ocirc;"  , "&otilde;" , "&ouml;"   , "&divide;"  ,;
                "&oslash;" , "&ugrave;" , "&uacute;" , "&ucirc;"  , "&uuml;"   , "&yacute;" , "&thorn;"  , "&yuml;"    }
   if ci != NIL 
   	nn := len(ci)
      for n := 1 to nn
         ch := asc(ci[n])
         if ch > 127 
         	co += t[ch-127]
         elseif ch == 38 
            co += "&amp;"
         elseif ch == 39 
            co += "&#39;"            
         elseif ch == 60
            co += "&lt;"
         elseif ch == 62
            co += "&gt;"
         elseif ch == 34
            co += "&quot;"         	
         else
         	co += ci[n]
         end
      next   
   end
   return co