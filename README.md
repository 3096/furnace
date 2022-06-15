# Ether Furnace
wismt texture tool

## License

    Ether Furnace
    Copyright (C) 2022  3096

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License along
    with this program. If not, see <https://www.gnu.org/licenses/>.

## Usage

    go run main.go <in wismt> <texture dir> <out wismt>

Under `<texture dir>` you would place your replacement texture files.

You must format your texture file names with <u><id.name.dds></u> (e.g. `00.PC079404_WAIST.dds`). The id will be used to identify the texture it replaces.

Along side the `wismt` file, you also need the `wimdo` file placed in the same directory. Both files need to be modified for the replaced textures to function correctly in game.

You can also replace using raw files by placing them in `<texture dir>/raw` directory, with filenames formatted in <u><index.whatever></u>.

Example:

    go run main.go ./test/formats_testdata/wismt/pc079404.wismt ./test/commands_testdata/msrd-replaced-textures ./output.wismt
