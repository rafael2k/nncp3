rm -fr nncp.html
MAKEINFO_OPTS="$MAKEINFO_OPTS --html --css-include style.css"
MAKEINFO_OPTS="$MAKEINFO_OPTS --set-customization-variable FORMAT_MENU=menu"
MAKEINFO_OPTS="$MAKEINFO_OPTS --set-customization-variable SHOW_TITLE=0"
MAKEINFO_OPTS="$MAKEINFO_OPTS --set-customization-variable DATE_IN_HEADER=1"
MAKEINFO_OPTS="$MAKEINFO_OPTS" . nncp.info.do
cp -r .well-known $3
