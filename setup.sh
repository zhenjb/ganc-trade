#!/bin/bash

# CONFIGURATION
SCRIPT_DIR="$HOME/scripts"
GANC_BIN="$SCRIPT_DIR/ganc"
COMPLETION_FILE="$SCRIPT_DIR/ganc_completion"
BASHRC="$HOME/.bashrc"
VERSION_APP=$(jq -r '.version' version.json)

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}[SETUP]        Installing Ganc v0.1.0 ...${NC}"

# 1. Create directory
mkdir -p "$SCRIPT_DIR"

# 2. Create GANC Engine
cat << 'EOF' > "$GANC_BIN"
#!/bin/bash

B='\033[0;34m'
G='\033[0;32m'
R='\033[0;31m'
Y='\033[1;33m'
NC='\033[0m'

COMMAND=$1
shift

case "$COMMAND" in
    test)
        DIR_PATH="test"
        FILE_NAME=""
        ARGS=""
        
        while [[ $# -gt 0 ]]; do
            case "$1" in
                -ob)
                    DIR_PATH="$DIR_PATH/order"
                    shift ;;
                -obs)
                    DIR_PATH="$DIR_PATH/obs"
                    shift ;;
                matching*|list*|market*|node*|smartc*)
                    FILE_NAME="${1#--}.sh"
                    shift ;;
                -help)
                    echo -e "${Y}USAGE${NC}"
                    echo -e "   ganc test -[flags] [args]"
                    echo -e "${Y}FLAGS${NC}"
                    echo -e "   -ob         Route to test/order (uses matching@...)"
                    echo -e "   -obs        Route to test/obs   (uses list@..., market@..., node@..., smartc@...)"
                    exit 0 ;;
                *)
                    ARGS+="$1 "
                    shift ;;
            esac
        done

        FULL_PATH="$DIR_PATH/$FILE_NAME"

        if [[ -f "$FULL_PATH" ]]; then
            echo -e "${G}[GANC]         Executing: bash $FULL_PATH $ARGS${NC}"
            bash "$FULL_PATH" $ARGS
        else
            echo -e "${R}[GANC]         Error: Program not found at $FULL_PATH${NC}"
            exit 1
        fi
        ;;

    chain)
        if [ -d "sw/ob" ]; then
            cd sw/ob
            echo -e "${G}[GANC]         Starting Ignite Chain...${NC}"
            ignite chain serve --reset-once
        else
            echo -e "${R}[GANC]         Error: Directory sw/ob not found.${NC}"
        fi
        ;;

    -v|version)
        VERSION=$(jq -r '.version' version.json)
        echo -e "${Y}Ganc CLI v.$VERSION${NC}"
        ;;

    -z|setup)
        echo -e "${G}[GANC]         Updating settings ...${NC}"
        bash exe.sh
        ;;

    -h|help)
        echo -e "${Y}USAGE${NC}"
        echo -e "   ganc [command] [flags] [args]"
        echo -e "${Y}COMMAND${NC}"
        echo -e "   test [flags] [args]      Program testing (use -ob or -obs)"
        echo -e "   chain                    Start ignite chain serve"
        echo -e "   version (-v)             Print version info"
        echo -e "   setup (-z)               Updating settings"
        echo -e "   help (-h)                Show this help"
        ;;
    
    *)
        echo -e "${R}Unknown command.${NC}"
        echo -e "Try: ${B}ganc help${NC}"
        ;;
esac
EOF

chmod +x "$GANC_BIN"

# 3. Create Smart Autocomplete (Updated for -ob and -obs)
cat << 'EOF' > "$COMPLETION_FILE"
_ganc_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    opts="test chain version help"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    if [[ "${COMP_WORDS[1]}" == "test" ]]; then
        case "$prev" in
            test)
                COMPREPLY=( $(compgen -W "-ob -obs -help" -- ${cur}) )
                ;;
            -ob)
                local files=$(ls test/order/*.sh 2>/dev/null | xargs -n1 basename | sed 's/\(.*\)\.sh/--\1/')
                COMPREPLY=( $(compgen -W "${files}" -- ${cur}) )
                ;;
            -obs)
                local files=$(ls test/obs/*.sh 2>/dev/null | xargs -n1 basename | sed 's/\(.*\)\.sh/--\1/')
                COMPREPLY=( $(compgen -W "${files}" -- ${cur}) )
                ;;
        esac
        return 0
    fi
}
complete -F _ganc_completions ganc
EOF

# 4. Finalizing .bashrc
if ! grep -q "$SCRIPT_DIR" "$BASHRC"; then
    echo -e "\n# GANC CLI" >> "$BASHRC"
    echo "export PATH=\"$SCRIPT_DIR:\$PATH\"" >> "$BASHRC"
    echo "source $COMPLETION_FILE" >> "$BASHRC"
fi

echo -e "${GREEN}[GANC]         Ganc v$VERSION_APP installation successful!${NC}"